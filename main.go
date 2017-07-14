package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"text/tabwriter"
	"text/template"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type entry struct {
	Name       string
	IngressIPs map[string]struct{}
}

const zoneTmpl = `; vim: set ft=bindzone :

` + "{{ range $zone := .ingresses }}{{ $zone.Name }}.{{ range $ip, $_ := $zone.IngressIPs }}\tIN\tA\t{{ $ip }}" + `
{{end}}
{{end}}`

var (
	inCluster = flag.Bool("incluster", false, "the client is running inside a kuberenetes cluster")
	once      = flag.Bool("once", false, "Write the file and then exit; do not watch for ingress changes")
	filepath  = flag.String("filepath", "zones/k8s-zones.cluster.local", "File location for zone file")
	command   = flag.String("command", "", "A command string to run any time the zone file is updated, useful for `rndc`")
	kubeHost  = flag.String("host", "", "The kubernetes API host; required if not run in-cluster")
	suffix    = flag.String("suffix", "", "The DNS suffix -- the controller will strip this from the end of ingress Host values if present")

	spaceRE = regexp.MustCompile("[[:space:]]+")
	ztpl    = template.Must(template.New("bind9").Parse(zoneTmpl))
)

func init() {
	flag.Parse()
}

func main() {
	var config *restclient.Config

	if *inCluster {
		var err error
		config, err = restclient.InClusterConfig()
		if err != nil {
			glog.Fatal("Error getting in-cluster config: ", err)
		}
	} else {
		if *kubeHost == "" {
			flag.Usage()
			glog.Fatal("Must run with -incluster (inside a k8s cluster) or provide a kubernetes host via -host")
		}
		config = &restclient.Config{
			Host: *kubeHost,
		}
	}

	cli, err := unversioned.New(config)
	if err != nil {
		glog.Fatal("Error creating API client: ", err)
	}

	if !*once {
		// Listen to SIGHUP and rewrite
		ch := make(chan os.Signal)
		go signal.Notify(ch, syscall.SIGHUP)

		go func() {
			for _ := range ch {
				createBindFile(cli)
			}
		}()

		glog.Fatal(watchIng(cli))
	}

	if err := createBindFile(cli); err != nil {
		glog.Fatal("Bind file creation error ", err)
	}
}

type ing struct {
	entry *entry
	orig  error
}

func (i ing) Error() string {
	return fmt.Sprintf("error with entry %s, ip %s: %s", i.entry.Name, i.entry.IngressIPs, i.orig)
}

func createBindFile(c *unversioned.Client) error {
	ingresses := map[string]*entry{}

	ings, err := c.Ingress("").List(api.ListOptions{})
	if err != nil {
		return ing{entry: nil, orig: err}
	}

	for _, ing := range ings.Items {
		glog.Info(ing.Name)

		ips := make(map[string]struct{}, len(ing.Status.LoadBalancer.Ingress))
		for i := range ing.Status.LoadBalancer.Ingress {
			ips[ing.Status.LoadBalancer.Ingress[i].IP] = struct{}{}
		}

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if strings.Contains(host, *suffix) && len(host) > len(*suffix)+1 {
				sfx := strings.Split(*suffix, ".")
				hst := strings.Split(host, ".")

				for len(sfx) > 0 && len(hst) > 0 && sfx[len(sfx)-1] == hst[len(hst)-1] {
					glog.V(2).Info(sfx, hst)
					sfx, hst = sfx[:len(sfx)-1], hst[:len(hst)-1]
				}
				host = strings.Join(hst, ".")
				glog.V(2).Info(host)
			}

			if host == "" {
				glog.V(2).Infof("Not adding empty host entry for ingress %s (was %s and suffix was %s)\n", ing.Name, rule.Host, *suffix)
				continue
			}

			if ing, ok := ingresses[host]; ok {
				for k := range ips {
					ing.IngressIPs[k] = struct{}{}
				}
				continue
			}

			ingresses[host] = &entry{
				Name:       host,
				IngressIPs: ips,
			}
		}
	}

	f, err := os.OpenFile(*filepath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(f, 2, 4, 1, ' ', 0)

	err = ztpl.Execute(tw, map[string]interface{}{"ingresses": ingresses})
	if err != nil {
		return err
	}

	err = tw.Flush()
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func watchIng(cli *unversioned.Client) error {
	for {
		w, err := cli.Extensions().Ingress("").Watch(api.ListOptions{})
		if err != nil {
			return fmt.Errorf("Watch error %s", err)
		}

		for evt := range w.ResultChan() {
			et := watch.EventType(evt.Type)
			if et != watch.Added && et != watch.Modified {
				continue
			}

			err = createBindFile(cli)
			if err != nil {
				return err
			}

			if *command != "" {
				s := spaceRE.Split(strings.Trim(*command, `"`), -1)
				glog.V(5).Info(s, len(s))
				cmd := exec.Command(s[0], s[1:]...)

				out, err := cmd.Output()
				os.Stdout.Write(out)
				if err != nil {

					switch err := err.(type) {
					case *exec.ExitError:
						glog.V(3).Infof("After %s, pid %d exited with success: '%t', and stderr:\n", err.UserTime(), err.Pid(), err.Success())
						os.Stderr.Write(err.Stderr)
					default:
						fmt.Fprintf(os.Stderr, "Encountered an unknown error: %s\n", err)
					}
				}
			}
		}

		glog.Warning("Result channel closed. Starting again.")
	}
}
