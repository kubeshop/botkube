# Kubectl executor

Kubectl is the Botkube executor plugin that allows you to run the Kubectl CLI commands directly from any communication platform.

## Configuration parameters

The configuration should be specified in the YAML format. Such parameters are supported:

```yaml
# Configures the default Namespace for executing Botkube `kubectl` commands. If not set, uses the 'default'.
defaultNamespace: "default"
# Configures the interactive kubectl command builder.
interactiveBuilder:
  allowed:
    # Configures which K8s namespace are displayed in namespace dropdown.
    # If not specified, plugin needs to have access to fetch all Namespaces, otherwise Namespace dropdown won't be visible at all.
    namespaces: ["default"]
    # Configures which `kubectl` methods are displayed in commands dropdown.
    verbs: ["api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top"]
    # Configures which K8s resource are displayed in resources dropdown.
    resources: [ "deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses" ]
```
