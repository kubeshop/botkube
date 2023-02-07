# Kubectl executor

Kubectl is the Botkube executor plugin that allows you to run the Kubectl CLI commands directly from any communication platform.

## Configuration parameters

The configuration should be specified in the YAML format. Such parameters are supported:

```yaml
commandBuilder:
  namespaces:
    - foo
    - bar
  commands:
    # -- Configures which `kubectl` methods are allowed.
    verbs: ["api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top"]
    # -- Configures which K8s resource are allowed.
    resources: ["deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses"]
```


Use `can-i` - `??`
