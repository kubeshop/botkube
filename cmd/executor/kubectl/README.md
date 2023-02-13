# Kubectl executor

Kubectl is the Botkube executor plugin that allows you to run the Kubectl CLI commands directly from any communication platform.

## Configuration parameters

The configuration should be specified in the YAML format. Such parameters are supported:

```yaml
defaultNamespace: "default"
interactiveBuilder:
  allowed:
    # Configures which K8s namespace are allowed. If not specified, plugin needs to have access to fetch all Namespaces, otherwise Namespace dropdown won't be visible. 
    namespaces:
      - foo
      - bar
    # Configures which `kubectl` methods are allowed.
    verbs: [ "api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top" ]
    # Configures which K8s resource are allowed.
    resources: [ "deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses" ]
```
