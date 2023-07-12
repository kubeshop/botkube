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
- RBAC is defaulted
- Plugins are sourced from Botkube repository

Use label selector to choose which Botkube pod you want to migrate. By default it's set to app=botkube.

Examples:

          $ botkube migrate --label app=botkube --instance-name botkube-slack     # Creates new Botkube Cloud instance with name botkube-slack and migrates pod with label app=botkube to it

	

```
botkube migrate [OPTIONS] [flags]
```

### Options

```
      --auto-upgrade                         Automatically upgrades Botkube instance without additional prompt
      --cfg-exporter-image-registry string   Config Exporter job image registry (default "ghcr.io")
      --cfg-exporter-image-repo string       Config Exporter job image repository (default "kubeshop/botkube-config-exporter")
      --cfg-exporter-image-tag string        Config Exporter job image tag (default "v9.99.9-dev")
      --cfg-exporter-poll-period duration    Config Exporter job poll period (default 1s)
      --cfg-exporter-timeout duration        Config Exporter job timeout (default 1m0s)
      --cloud-api-url string                 Botkube Cloud API URL (default "https://api.botkube.io/graphql")
      --cloud-dashboard-url string           Botkube Cloud URL (default "https://app.botkube.io")
  -h, --help                                 help for migrate
      --instance-name string                 Botkube Cloud Instance name that will be created
  -l, --label string                         Label of Botkube pod (default "app=botkube")
  -n, --namespace string                     Namespace of Botkube pod (default "botkube")
  -q, --skip-connect                         Skips connecting to Botkube Cloud after migration
      --skip-open-browser                    Skips opening web browser after migration
      --token string                         Botkube Cloud authentication token
```

### Options inherited from parent commands

```
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube](botkube.md)	 - Botkube CLI

