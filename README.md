# Kube-named-zones (700-MB image version)

Bind9/named automatic zone file generation from kubernetes ingresses

To deploy to kubernetes, you probably want to alter the default values
(especially for the DNS `-suffix`), but I've provided a basic deployment
manifest.
```bash
kubectl create -f deployment.yml
```

Or run ad-hoc
```
Usage of kube-named-zones:
  -command rndc
    	A command string to run any time the zone file is updated, useful for rndc
  -filepath string
    	File location for zone file (default "zones/k8s-zones.cluster.local")
  -host string
    	The kubernetes API host; required if not run in-cluster
  -incluster
    	the client is running inside a kuberenetes cluster
  -once
    	Write the file and then exit; do not watch for ingress changes
  -suffix string
    	The DNS suffix -- the controller will strip this from the end of ingress Host values if present
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -alsologtostderr
    	log to standard error as well as files
  -logtostderr
    	log to standard error instead of files
```
