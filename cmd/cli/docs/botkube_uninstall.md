---
title: botkube uninstall
---

## botkube uninstall

uninstall Botkube from cluster

### Synopsis

Use this command to uninstall the Botkube agent.

```
botkube uninstall [OPTIONS] [flags]
```

### Examples

```
# Uninstall default Botkube Helm release
botkube uninstall

# Uninstall specific Botkube Helm release
botkube uninstall --release-name botkube-dev
```

### Options

```
  -y, --auto-approve          Skips interactive approval for deletion.
      --cascade string        Must be "background", "orphan", or "foreground". Selects the deletion cascading strategy for the dependents. Defaults to background. (default "background")
      --description string    add a custom description
      --dry-run               Simulate an uninstallation
  -h, --help                  help for uninstall
      --keep-history          remove all associated resources and mark the release as deleted, but retain the release history
      --kubeconfig string     Paths to a kubeconfig. Only required if out-of-cluster.
      --namespace string      Botkube namespace. (default "botkube")
      --no-hooks              prevent hooks from running during uninstallation
      --release-name string   Botkube Helm release name. (default "botkube")
      --timeout duration      time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)
      --wait                  if set, will wait until all the resources are deleted before returning. It will wait for as long as --timeout (default true)
```

### Options inherited from parent commands

```
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube](botkube.md)	 - Botkube CLI

