# ConfigMap watcher source

Kubernetes ConfigMap watcher is an example Botkube source plugin used during [e2e tests](../../../test/e2e). It's not meant for production usage.

## Configuration parameters

The configuration should be specified in the YAML format. Such parameters are supported:

```yaml
configMap:
  name: cm-map-watcher # config map name to react to
  namespace: botkube  # config map namespace
```
