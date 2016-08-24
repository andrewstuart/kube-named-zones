package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

	if err := createBindFile(cli); err != nil {
		log.Fatal("Bind file creation error ", err)
	}

	// log.Fatal(watchIng(cli))
}

type ing struct {
	entry *entry
	orig  error
}

func (i ing) Error() string {
	return fmt.Sprintf("error with entry %s, ip %s: %s", i.entry.Name, i.entry.IngressIPs, i.orig)
}

func createBindFile(c *unversioned.Client) error {
	nss, err := c.Namespaces().List(api.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	ingresses := []entry{}

	for _, ns := range nss.Items {
		log.Println(ns.Name)
		ings, err := c.Ingress(ns.Namespace).List(api.ListOptions{})
		if err != nil {
			return ing{entry: nil, orig: err}
		}

		for _, ing := range ings.Items {
			ips := make([]string, len(ing.Status.LoadBalancer.Ingress))
			for i := range ing.Status.LoadBalancer.Ingress {
				ips[i] = ing.Status.LoadBalancer.Ingress[i].IP
			}

			for _, rule := range ing.Spec.Rules {
				ingresses = append(ingresses, entry{
					Name:       rule.Host,
					IngressIPs: ips,
				})
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

	tw.Flush()

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
		}

		log.Println("Result channel closed. Starting again.")
	}
}
