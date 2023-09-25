---
title: botkube
---

## botkube

Botkube CLI

### Synopsis

botkube - Botkube CLI

A utility that simplifies working with Botkube.

Quick Start:

    $ botkube install                              # Install Botkube
    $ botkube uninstall                            # Uninstall Botkube

Botkube Cloud:

    $ botkube login                                # Login into Botkube Cloud
    $ botkube migrate                              # Automatically migrates Open Source installation into Botkube Cloud
    

```
botkube [flags]
```

### Options

```
  -h, --help                          help for botkube
  -v, --verbose int/string[=simple]   Prints more verbose output. Allowed values: 0 - disable, 1 - simple, 2 - trace (default 0 - disable)
```

### SEE ALSO

* [botkube config](botkube_config.md)	 - This command consists of multiple subcommands for working with Botkube configuration
* [botkube install](botkube_install.md)	 - install or upgrade Botkube in k8s cluster
* [botkube login](botkube_login.md)	 - Login to a Botkube Cloud
* [botkube migrate](botkube_migrate.md)	 - Automatically migrates Botkube installation into Botkube Cloud
* [botkube telemetry](botkube_telemetry.md)	 - This command consists of subcommands to disable or enable telemetry
* [botkube uninstall](botkube_uninstall.md)	 - uninstall Botkube from cluster
* [botkube version](botkube_version.md)	 - Print the CLI version

