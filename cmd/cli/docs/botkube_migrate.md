---
title: botkube migrate
---

## botkube migrate

Automatically migrates Botkube installation into Botkube Cloud

### Synopsis

Automatically migrates Botkube installation to Botkube Cloud.
This command will create a new Botkube Cloud instance based on your existing Botkube configuration, and upgrade your Botkube installation to use the remote configuration.

Supported Botkube bot platforms for migration:
- Socket Slack
- Discord
- Mattermost

Limitations:
- Plugins are sourced from Botkube repository

Use label selector to choose which Botkube pod you want to migrate. By default it's set to app=botkube.

Examples:

          $ botkube migrate --label app=botkube --instance-name botkube-slack     # Creates new Botkube Cloud instance with name botkube-slack and migrates pod with label app=botkube to it

	

```
botkube migrate [OPTIONS] [flags]
```

### Options

```
  -y, --auto-approve                         Skips interactive approval for upgrading Botkube installation.
      --cfg-exporter-image-registry string   Registry for the Config Exporter job image (default "ghcr.io")
      --cfg-exporter-image-repo string       Repository for the Config Exporter job image (default "kubeshop/botkube-config-exporter")
      --cfg-exporter-image-tag string        Tag of the Config Exporter job image (default "v9.99.9-dev")
      --cfg-exporter-poll-period duration    Interval used to check if Config Exporter job was finished (default 1s)
      --cfg-exporter-timeout duration        Maximum execution time for the Config Exporter job (default 1m0s)
      --cloud-api-url string                 Botkube Cloud API URL (default "https://api.botkube.io/graphql")
      --cloud-dashboard-url string           Botkube Cloud URL (default "https://app.botkube.io")
      --cloud-env-api-key string             API key environment variable name specified under Deployment for cloud installation. (default "CONFIG_PROVIDER_API_KEY")
      --cloud-env-endpoint string            Endpoint environment variable name specified under Deployment for cloud installation. (default "CONFIG_PROVIDER_ENDPOINT")
      --cloud-env-id string                  Identifier environment variable name specified under Deployment for cloud installation. (default "CONFIG_PROVIDER_IDENTIFIER")
  -h, --help                                 help for migrate
      --image-tag string                     Botkube image tag, e.g. "latest" or "v1.7.0"
      --instance-name string                 Botkube Cloud Instance name that will be created
      --kubeconfig string                    Paths to a kubeconfig. Only required if out-of-cluster.
      --kubecontext string                   The name of the kubeconfig context to use.
  -l, --label string                         Label used for identifying the Botkube pod (default "app=botkube")
  -n, --namespace string                     Namespace of Botkube pod (default "botkube")
  -q, --skip-connect                         Skips connecting to Botkube Cloud after migration
      --skip-open-browser                    Skips opening web browser after migration
      --timeout duration                     Maximum time during which the Botkube installation is being watched, where "0" means "infinite". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". (default 10m0s)
      --token string                         Botkube Cloud authentication token
  -w, --watch --timeout                      Watches the status of the Botkube installation until it finish or the defined --timeout occurs. (default true)
```

### Options inherited from parent commands

```
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube](botkube.md)	 - Botkube CLI

