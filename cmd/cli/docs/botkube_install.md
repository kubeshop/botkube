---
title: botkube install
---

## botkube install

install or upgrade Botkube in k8s cluster

### Synopsis

Use this command to install or upgrade the Botkube agent.

```
botkube install [OPTIONS] [flags]
```

### Examples

```
# Install latest stable Botkube version
botkube install

# Install Botkube 0.1.0 version
botkube install --version 0.1.0

# Install Botkube from local git repository. Needs to be run from the main directory.
botkube install --repo @local
```

### Options

```
      --atomic                       If set, process rolls back changes made in case of failed install/upgrade. The --wait flag will be set automatically if --atomic is used
  -y, --auto-approve                 Skips interactive approval when upgrade is required.
      --chart-name string            Botkube Helm chart name. (default "botkube")
      --dependency-update            Update dependencies if they are missing before installing the chart
      --description string           add a custom description
      --disable-openapi-validation   If set, it will not validate rendered templates against the Kubernetes OpenAPI Schema
      --dry-run                      Simulate an installation
      --force                        Force resource updates through a replacement strategy
  -h, --help                         help for install
      --kubeconfig string            Paths to a kubeconfig. Only required if out-of-cluster.
      --kubecontext string           The name of the kubeconfig context to use.
      --namespace string             Botkube installation namespace. (default "botkube")
      --no-hooks                     Disable pre/post install/upgrade hooks
      --release-name string          Botkube Helm chart release name. (default "botkube")
      --render-subchart-notes        If set, render subchart notes along with the parent
      --repo string                  Botkube Helm chart repository location. It can be relative path to current working directory or URL. Use @stable tag to select repository which holds the stable Helm chart versions. (default "https://charts.botkube.io/")
      --reset-values                 When upgrading, reset the values to the ones built into the chart
      --reuse-values                 When upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored
      --set stringArray              Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
      --set-file stringArray         Set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)
      --set-json stringArray         Set JSON values on the command line (can specify multiple or separate values with commas: key1=jsonval1,key2=jsonval2)
      --set-literal stringArray      Set a literal STRING value on the command line
      --set-string stringArray       Set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
      --skip-crds                    If set, no CRDs will be installed.
      --timeout duration             Maximum time during which the Botkube installation is being watched, where "0" means "infinite". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". (default 10m0s)
  -f, --values strings               Specify values in a YAML file or a URL (can specify multiple)
      --version string               Botkube version. Possible values @latest, 1.2.0, ... (default "@latest")
  -w, --watch --timeout              Watches the status of the Botkube installation until it finish or the defined --timeout occurs. (default true)
```

### Options inherited from parent commands

```
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube](botkube.md)	 - Botkube CLI

