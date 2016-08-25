package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/tabwriter"
	"text/template"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type entry struct {
	Name       string
	IngressIPs []string
}

const zoneTmpl = `; vim: set ft=bindzone :

` + "{{ range $zone := .ingresses }}{{ $zone.Name }}{{ range $ip := $zone.IngressIPs }}\tIN\tA\t{{ $ip }}" + `
{{end}}
{{end}}`

var (
	inCluster = flag.Bool("incluster", false, "the client is running inside a kuberenetes cluster")
	filepath  = flag.String("filepath", "zones/k8s-zones.cluster.local", "File location for zone file")
	command   = flag.String("command", "", "A command to run any time the zone file is updated")
	suffix    = flag.String("suffix", "astuart.co", "The DNS suffix")

	spaceRE = regexp.MustCompile("[[:space:]]+")

	ztpl = template.Must(template.New("bind9").Parse(zoneTmpl))
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
			log.Fatal(err)
		}
	} else {
		config = &restclient.Config{
			Host: "http://desk.astuart.co:8080",
		}
	}

	cli, err := unversioned.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// if err := createBindFile(cli); err != nil {
	// 	log.Fatal("Bind file creation error ", err)
	// }

	log.Fatal(watchIng(cli))
}

type ing struct {
	entry *entry
	orig  error
}

func (i ing) Error() string {
	return fmt.Sprintf("error with entry %s, ip %s: %s", i.entry.Name, i.entry.IngressIPs, i.orig)
}

func createBindFile(c *unversioned.Client) error {
	ingresses := []entry{}

	ings, err := c.Ingress("").List(api.ListOptions{})
	if err != nil {
		return ing{entry: nil, orig: err}
	}

	for _, ing := range ings.Items {
		log.Println(ing.Name)

		ips := make([]string, len(ing.Status.LoadBalancer.Ingress))
		for i := range ing.Status.LoadBalancer.Ingress {
			ips[i] = ing.Status.LoadBalancer.Ingress[i].IP
		}

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if strings.Contains(host, *suffix) && len(host) > len(*suffix)+1 {
				host = host[:len(host)-1-len(*suffix)]
			}

			ingresses = append(ingresses, entry{
				Name:       host,
				IngressIPs: ips,
			})
		}
	}
	log.Println()

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
				log.Println(s, len(s))
				cmd := exec.Command(s[0], s[1:]...)

				out, err := cmd.Output()
				os.Stdout.Write(out)
				if err != nil {

					switch err := err.(type) {
					case *exec.ExitError:
						fmt.Fprintf(os.Stderr, "After %s, pid %d exited with success: '%t', and stderr:\n", err.UserTime(), err.Pid(), err.Success())
						os.Stderr.Write(err.Stderr)
					default:
						fmt.Fprintf(os.Stderr, "Encountered an unknown error: %s\n", err)
					}
				}
			}
		}

		log.Println("Result channel closed. Starting again.")
	}
}
