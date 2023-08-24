---
title: botkube config get
---

## botkube config get

Displays Botkube configuration

```
botkube config get [flags]
```

### Examples

```
# Show configuration for currently installed Botkube
botkube config get

# Show configuration in JSON format
botkube config get -ojson

# Save configuration in file
botkube config get > config.yaml

```

### Options

```
      --cfg-exporter-image-registry string   Registry for the Config Exporter job image (default "ghcr.io")
      --cfg-exporter-image-repo string       Repository for the Config Exporter job image (default "kubeshop/botkube-config-exporter")
      --cfg-exporter-image-tag string        Tag of the Config Exporter job image (default "v9.99.9-dev")
      --cfg-exporter-poll-period duration    Interval used to check if Config Exporter job was finished (default 1s)
      --cfg-exporter-timeout duration        Maximum execution time for the Config Exporter job (default 1m0s)
  -h, --help                                 help for get
  -l, --label string                         Label used for identifying the Botkube pod (default "app=botkube")
  -n, --namespace string                     Namespace of Botkube pod (default "botkube")
      --omit-empty-values                    Omits empty keys from printed configuration (default true)
  -o, --output string                        Output format. One of: json | yaml (default "yaml")
```

### Options inherited from parent commands

```
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube config](botkube_config.md)	 - This command consists of multiple subcommands for working with Botkube configuration

