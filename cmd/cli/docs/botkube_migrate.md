---
title: botkube migrate
---

## botkube migrate

Automatically migrates Botkube installation into Botkube Cloud

### Synopsis

Automatically migrates Botkube installation into Botkube Cloud

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
      --cloud-api-url string         Botkube Cloud API URL (default "https://api.botkube.io")
      --cloud-dashboard-url string   Botkube Cloud URL (default "https://app.botkube.io")
  -h, --help                         help for migrate
      --instance-name string         Botkube Cloud Instance name that will be created
  -l, --label string                 Label of Botkube pod (default "app=botkube")
  -n, --namespace string             Namespace of Botkube pod (default "botkube")
  -q, --skip-connect                 Skips connecting to Botkube Cloud after migration
      --token string                 Botkube Cloud authentication token
```

### SEE ALSO

* [botkube](botkube.md)	 - Botkube Cloud CLI

