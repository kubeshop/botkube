package kubectl

import (
	"github.com/MakeNowJust/heredoc"
)

func help() string {
	return heredoc.Doc(`
		The official Botkube plugin for the Kubectl CLI.

		Usage:
		  kubectl [flags] [options]

		Supported Commands:
		  expose          Take a replication controller, service, deployment or pod and expose it as a new Kubernetes service
		  run             Run a particular image on the cluster
		  set             Set specific features on objects

		  explain         Get documentation for a resource
		  get             Display one or many resources
		  delete          Delete resources by file names, stdin, resources and names, or by resources and label selector

		  rollout         Manage the rollout of a resource
		  scale           Set a new size for a deployment, replica set, or replication controller
		  autoscale       Auto-scale a deployment, replica set, stateful set, or replication controller

		  certificate     Modify certificate resources.
		  cluster-info    Display cluster information
		  top             Display resource (CPU/memory) usage
		  cordon          Mark node as unschedulable
		  uncordon        Mark node as schedulable
		  drain           Drain node in preparation for maintenance
		  taint           Update the taints on one or more nodes

		  describe        Show details of a specific resource or group of resources
		  logs            Print the logs for a container in a pod
		  exec            Execute a command in a container
		  auth            Inspect authorization

		  patch           Update fields of a resource
		  replace         Replace a resource by file name or stdin

		  label           Update the labels on a resource
		  annotate        Update the annotations on a resource

		  api-resources   Print the supported API resources on the server
		  api-versions    Print the supported API versions on the server, in the form of "group/version"
		  version         Print the client and server version information
	`)
}
