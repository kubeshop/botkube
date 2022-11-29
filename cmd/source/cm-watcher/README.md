# ConfigMap watcher source

ConfigMap watcher source is the example Botkube source used during [e2e tests](../../../test/e2e).

## Configuration parameters

The configuration should be specified in YAML format. Such parameters are supported:

```yaml
configMap:
  name: cm-map-watcher # config map name to react to
  namespace: botkube  # config map namespace
```
