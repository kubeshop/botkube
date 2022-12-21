# Helm executor

Helm is the Botkube executor plugin that allows you to run the Helm CLI commands directly from any communication platform.

## Configuration parameters

The configuration should be specified in the YAML format. Such parameters are supported:

```yaml
helmDriver: "secret", # Allowed values are configmap, secret, memory.
helmCacheDir: "/tmp/helm/.cache",
helmConfigDir: "/tmp/helm/",
```
