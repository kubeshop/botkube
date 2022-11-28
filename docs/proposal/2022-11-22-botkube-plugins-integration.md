## Integrate plugins into Botkube

Created on 2022-11-22 by Mateusz Szostok ([@mszostok](https://github.com/mszostok))

| Status     |
|------------|
| `ACCEPTED` |

<!-- TOC -->
  * [New syntax](#new-syntax)
  * [Use cases](#use-cases)
    * [Defining list of plugin repositories and enabling a given plugin](#defining-list-of-plugin-repositories-and-enabling-a-given-plugin)
    * [Defining executors aliases](#defining-executors-aliases)
    * [Validating plugin configuration](#validating-plugin-configuration)
      * [Plugin configuration](#plugin-configuration)
    * [Passing configuration to a given plugin](#passing-configuration-to-a-given-plugin)
    * [Refreshing already loaded plugins](#refreshing-already-loaded-plugins)
    * [Supporting multi OS and architectures](#supporting-multi-os-and-architectures)
    * [Releasing Botkube plugins](#releasing-botkube-plugins)
  * [Implementation details](#implementation-details)
    * [E2E testing](#e2e-testing)
    * [How plugins are stored](#how-plugins-are-stored)
    * [Do we want to decouple the interfaces between plugin Go implementation and gRPC?](#do-we-want-to-decouple-the-interfaces-between-plugin-go-implementation-and-grpc)
    * [Do we want to have a separate go mod for each executor/source?](#do-we-want-to-have-a-separate-go-mod-for-each-executorsource)
    * [Botkube directory structure](#botkube-directory-structure)
  * [Alternatives](#alternatives)
    * [The `plugins` object syntax](#the-plugins-object-syntax)
    * [How a given plugins is configured](#how-a-given-plugins-is-configured)
<!-- TOC -->

## Motivation

This proposal describe how the [plugin system](2022-09-28-botkube-plugin-system.md) can be integrated with the Botkube core. It describes both the configuration syntax changes and expected behavior.

## New syntax

This section describes the necessary changes in the syntax. **It's backward compatible.**

1. A new `plugins` property. It allows to specify a list of repository from where the plugins can be downloaded.
   ```yaml
   plugins:
     cacheDir: "/tmp"
     repositories:
       botkube:
         url: https://plugins.botkube.io/botkube.yaml
       huseyinbabal:
         url: https://raw.githubusercontent.com/huseyinbabal/botkube-plugins/main/index.json
   ```

2. The `executors` definition now can refer to the executor plugins specified in a given repository.
   ```yaml
   executors:
     'plugin-based':
       botkube/kubectl@v1.0.0:     # <repo>/<plugin>[@<version>] is syntax for plugin based executors. If version is not provided, the latest version from repository is used.
         enabled: true      # if not enabled we don't download and start a given plugin
         config:            # plugin specific configuration
           namespaces:
             include: ["botkube", "default","ambassador"]
           commands:
             verbs: ["get","logs","top"]
             resources: ["pods","deployments","nodes","configmap"]
   ```

3. The `sources` definition now can refer to the source plugins specified in a given repository.
   ```yaml
   sources:
     'plugin-based':
       botkube/kubernetes@v1.0.0:
         enabled: true
         config:
           recommendations:
             pod:
               noLatestImageTag: true
               labelsSet: true
   ```

## Use cases

This section describes example configurations that enable the requested use-cases.

### Defining list of plugin repositories and enabling a given plugin

We introduce a new `plugins` object to Botkube configuration syntax. It holds the list of the plugins repositories. Each repository has own name to prevent conflict with the same plugins name, e.g. both provides `kubectl` plugin.

To enable a given plugin you need to specify it under the `executors` or `sources` property. In the first phase, we only allow to enable and disable plugins using Botkube configuration. Later we can introduce `@Botkube enable [executor|source] NAME` and `@Botkube disable [executor|source] NAME`. Disabling and enabling plugins should be possible in the Botkube runtime without restarting it.

**Syntax:**

```yaml
executors:
  'plugin-based':
    botkube/kubectl@v1.0.0:    # <repo>/<plugin>[@<version>] is syntax for plugin based executors
      enabled: true            # if not enabled we don't download and start a given plugin
      config:                  # plugin specific configuration
        # ...
sources:
  'plugin-based':
    botkube/kubernetes:       # if version is not specified, the latest version is used.
      enabled: true
      config:
        # ...
```

In this way you can:

1. Register multiple plugins repositories.
2. Enable different plugin version from the same repository.
3. Enable the same plugin (e.g. `kubectl`) from different repositories.

### Defining executors aliases

Currently, the aliases are built-in into Botkube core logic. We can allow to specify them when enabling a given plugin:

```yaml
executors:
  'plugin-based':
    botkube/kubectl@v1.0.0:
      enabled: true
      aliases: [ "kc", "k" ]
```

With such configuration, the `kubectl` executor plugin can be used with the following command prefixes: `kubectl`, `kc` and `k`.

### Validating plugin configuration

We introduce a basic validation to make sure that:

1. Entries define in the index files don't conflict with each other - having the same `name`, `type` and `version`:

   <details>
     <summary>Example</summary>

   ```yaml
   entries:
     - name: "kubectl"
       type: "executor"
       description: "Kubectl executor plugin."
       version: "v1.0.0"
       urls:
         - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/source_kubernetes-darwin-amd64
           platform:
             architecture: amd64
             os: darwin
     - name: "kubectl"
       type: "executor"
       description: "Kubectl executor plugin."
       version: "v1.0.0"
       urls:
         - url: https://github.com/kubeshop/botkube/releases/download/v0.16.0/source_kubernetes-darwin-amd64
           platform:
             architecture: amd64
             os: darwin
   ```
   </details>

2. Enabled executors don't conflict with each other:
   <details>
     <summary>Example</summary>

   ```yaml
   executors:
     'plugin-based':
       botkube/kubectl:
         enabled: true
         config:
           namespaces:
             include: ["botkube", "default","ambassador"]
           commands:
             verbs: ["get","logs","top"]
             resources: ["pods","deployments","nodes","configmap"]
       mszostok/kubectl:  # not allowed as it is conflicting with already registered 'kubectl' from the Botkube repository.
         enabled: true
         config:
           commands:
             verbs: ["get","logs","top"]
   ```
   ```yaml
   executors:
     'plugin-based':
       botkube/kubectl@v1.0.0:
         enabled: true
         config:
           namespaces:
             include: ["botkube", "default","ambassador"]
           commands:
             verbs: ["get","logs","top"]
             resources: ["pods","deployments","nodes","configmap"]
       botkube/kubectl@v1.2.0:  # not allowed as it is conflicting with already registered 'kubectl' from the Botkube repository in different version.
         enabled: true
         config:
           commands:
             verbs: ["get","logs","top"]
   ```
   </details>

3. We cannot have bindings to the same executor but from different repositories or versions:
   <details>
     <summary>Example</summary>

   ```yaml
   communications:
     default-group:
       socketSlack:
         enabled: true
         channels:
           default:
             name: botkubers
             bindings:
               executors:            # such binding is not allowed as it has conflicting 'kubectl' executor from different repositories
                 - kubectl-read-only
                 - kubectl-pods-rw
   executors:
     'kubectl-read-only':
       botkube/kubectl:
         enabled: true
         config:
           # ...
     'kubectl-pods-rw':
       mszostok/kubectl:
         enabled: true
         config:
           # ...
     'kubectl-deploy-rw':
       botkube/kubectl@v1.0.0:
         enabled: true
         config:
           # ...
   ```
   </details>

#### Plugin configuration

The plugin specific configuration is not validated by Botkube. It's plugins responsibility to do that. In the future, we can allow specifying JSON Schema for each plugin defined in the index file, for example:

```yaml
entries:
  - name: "kubectl"
    type: "executor"
    description: "Kubectl executor plugin."
    urls:
      - https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-amd64
    jsonSchema:
      value: |-
        {
          "$schema": "http://json-schema.org/draft-07/schema",
          "type": "object",
          "required": [
              "version"
          ],
          "properties": {
            "version": {
              "$id": "#/properties/version",
              "type": "string",
              "minLength": 5,
              "pattern": "^(?:0|[1-9]\\d*)\\.(?:0|[1-9]\\d*)\\.(?:0|[1-9]\\d*)$",
              "title": "Kubernetes version",
              "description": "Kubernetes version",
              "default": "",
              "examples": [
                  "1.19.0"
              ]
            }
          },
          "additionalProperties": true
        }
```

This will be used by Botkube to validate that configuration defined by user is valid before even starting a given plugin.

### Passing configuration to a given plugin

Current Botkube implementation allows you to specify different executor and sources configuration and bind them to a single channel. Later they are merged together based on the given business logic.

To support the same experience for plugins, we need to pass a list of configuration each time we run the:

- `Execute` method for executor plugins, invoked when user runs a given command,
- `Source` method for source plugins, invoked during Botkube startup.

We pass an array of strings, so later they can be unmarshalled by a given plugin and merged based on custom business logic.

We don't support configuration validation inside Botkube. See validating [plugins configuration](#plugin-configuration) for more information.

The allowed configuration parameters will be described on the `docs.botkube.io` site in the same way that we did for `kubectl` executor and `kuberneetes` source.

### Refreshing already loaded plugins

Once the Botkube starts it caches:

- all index files for defined plugins repositories
- all enabled plugins binaries

We use the `emptyDir` so the data is not removed as long as the Pod is not rescheduled to a different Node. To force download, you need to run:

```bash
kubectl exec -it $(kubectl get po -l app=botkube -n botkube -oname) -- rm -rf /tmp
kubectl delete po -l app=botkube -n botkube
```

Later we can provide a dedicated `@Botkube` command to simplify refreshing downloaded plugins. However, in the happy path scenarios no one should replace already released binaries.

### Supporting multi OS and architectures

We want to run plugins on different Kubernetes distros and also locally for development purposes. As a result, we need to support different operating systems and different architectures.

We enforce that plugin binaries have a suffix with a given pattern: `{os}_{arch}[.exe]`. For example:

- `botkube_executor_helm_linux_amd64`
- `botkube_source_prometheus_darwin_amd64`

Botkube uses the `runtime.GOOS` and `runtime.GOARCH` variables to determine on which system Botkube is running. It downloads respected binary by searching a proper link in plugin index file:

```yaml
entries:
  - name: "kubectl"
    type: "executor"
    description: "Kubectl executor plugin."
    version: "v1.0.0"
    urls:
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-amd64
        platform:
          architecture: amd64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-arm64
        platform:
          architecture: arm64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-amd64
        platform:
          architecture: amd64
          os: linux
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-arm64
        platform:
          architecture: arm64
          os: linux
```

In the first phase:

- we support only Linux and macOS. Support for more platforms will be added when explicitly requested.
- we support only HTTP servers. Support for different protocols will be added when explicit requested. We can utilize the [`hashicorp/go-getter`](https://github.com/hashicorp/go-getter) library to introduce support for downloading plugins directly from Git, Mercurial, Amazon S3, Google GCP. With such approach we will simply support a new syntax, e.g. for Git it will be `github.com/kubeshop/botkube-plugins?ref={tag/commit/branch}`

### Releasing Botkube plugins

As a part of 0.17 release, such binaries should be added to the GitHub release assets:

- `plugins-index.yaml`
- `source_kubernetes-darwin-amd64`
- `source_kubernetes-linux-amd64`
- `executor_kubectl-darwin-amd64`
- `executor_kubectl-linux-amd64`

Where the `plugins-index.yaml` is defined as follows:

```yaml
entries:
  - name: "kubectl"
    type: "executor"
    description: "Kubectl executor plugin."
    version: "v1.0.0"
    urls:
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-amd64
        platform:
          architecture: amd64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-arm64
        platform:
          architecture: arm64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-amd64
        platform:
          architecture: amd64
          os: linux
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-arm64
        platform:
          architecture: arm64
          os: linux
  - name: "kubernetes"
    type: "source"
    description: "Kubernetes source plugin."
    version: "v1.0.0"
    urls:
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/source_kubernetes-darwin-amd64
        platform:
          architecture: amd64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/source_kubernetes-darwin-arm64
        platform:
          architecture: arm64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/source_kubernetes-linux-amd64
        platform:
          architecture: amd64
          os: linux
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/source_kubernetes-linux-arm64
        platform:
          architecture: arm64
          os: linux
```

The index file can be generated automatically just by scanning directory where plugins binaries are stored.

## Implementation details

### E2E testing

Botkube fetches the plugins index and binaries when it's started. However, if they are already present, Botkube doesn't take any action. We could leverage that and simply built all plugins binaries and include them in Docker image with in a proper directories.
Unfortunately, this approach doesn't test whether the download mechanism works properly. To overcome this issue we can still build all plugins on pull-request but instead of embedding them into Docker images we can start a simple static file server. This server can be later accessed by a running Pod via `host.k3d.internal` DNS which enables access to host system. This feature is natively supported by [k3d](https://k3d.io/v5.0.1/faq/faq/#how-to-access-services-like-a-database-running-on-my-docker-host-machine).

### How plugins are stored

Once downloaded, such folder structure is created:

```plaintext
cache-dir
├── botkube                  # repository name
│  ├── executor_v0.1.0_echo  # executor binary with patter 'executor_<version>_<name>'
│  └── source_v0.1.0_echo    # source binary with patter 'source_<version>_<name>'
└── botkube.yaml             # repository index file
```

Before downloading index file or plugin binaries, Botkube checks whether a given file already exist. If yes, no action is taken.

### Decoupling interfaces between Go implementation and gRPC

For now, we will reuse the generated struct from the Protocol Buffers.

### Separate Go modules for each plugin

For now, we'll keep the shared dependencies as long as we don't extract the `kubectl` executor and `kubernetes` source as external plugins. Later we can revisit this decision as it can bring additional benefits, such as shorter build time and reduced binary size.

### Botkube directory structure

```plaintext
.
├── bin  # git ignored folder. Here are installed the protoc, grpc plugins, etc.
├── cmd
│  ├── botkube  # Botube Core
│  ├── executor # here we add entrypoints to our executors
│  │  └── echo
│  └── source   # here we add entrypoints to our sources
│     └── kubernetes
├── internal
│  ├── plugin   # here is internal logic to manage plugins within Botkube Core.
│  └── ...
├── pkg
│  ├── api      # Botkube public API that can be imported also by some 3rd party libs/apps.
│  │  ├── executor
│  │  └── source
│  └── ...
└── proto        # here are the Protocol buffers API that can be used to generate clients in different languages. It's on the root as it's Botkube not Go specific files. We generate Go client/server into `pkg/api/{executor|source}`, so it can be use by other 3rd plugins.
```

## Alternatives

Other approaches that were considered but rejected because of the complexity.

<details>
  <summary>Discarded alternative</summary>

### The `plugins` object syntax

Configuration defined in `values.yaml`

```yaml
plugins:
  repositories:
    botkube: "https://github.com/kubeshop/botkube/releases/download/0.17.0/plugins-index.yaml"
    mszostok: "https://github.com/mszostok/botkube-plugins/releases/download/latest/index.yaml"
  # alternative, which is more extensible but doesn't allow to enforce unique name OOTB
  repositories:
    - name: botkube
      url: "https://github.com/kubeshop/botkube/releases/download/0.17.0/plugins-index.yaml"
    - name: mszostok
      url: "https://github.com/mszostok/botkube-plugins/releases/download/latest/index.yaml"
  # alternative, which is more extensible and allow to enforce unique name OOTB
  repositories:
    botkube:
      url: "https://github.com/kubeshop/botkube/releases/download/0.17.0/plugins-index.yaml"
    mszostok:
      url: "https://github.com/mszostok/botkube-plugins/releases/download/latest/index.yaml"

  enabled: # or 'install:'
    - name: botkube/kubernetes
      version: v0.17.0
    - name: mszostok/kubectl
      # version not provided, so use latest
```

We could also introduce the CRD, for example:

```yaml
apiVersion: plugins.botkube.io/v1alpha1
kind: ClusterPluginConfiguration
metadata:
  name: bk-plugins
spec:
  repositories:
    official: "https://github.com/kubeshop/botkube/releases/download/0.17.0/plugins-index.yaml"
  enabled:
    - name: official/kubernetes
      version: v0.17.0
```

### How a given plugins is configured

- Create dedicated instance (subprocess) for each different plugin config. This won't scale well.
- Passing configuration data when starting a given plugin:
  - Pass as serialized JSON/YAML using cmd flags.
  - Save to file and specify flag with path location.
- Add additional method like `Initialize()/SetConfig()`

### Defining executors aliases

One option is to define them in the index file for a given executor or source:

```yaml
entries:
  - name: "kubectl"
    type: "executor"
    description: "Kubectl executor plugin."
    version: "v1.0.0"
    aliases: [ "kc", "k" ]
    urls:
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-amd64
        platform:
          architecture: amd64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-darwin-arm64
        platform:
          architecture: arm64
          os: darwin
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-amd64
        platform:
          architecture: amd64
          os: linux
      - url: https://github.com/kubeshop/botkube/releases/download/v0.17.0/executor_kubectl-linux-arm64
        platform:
          architecture: arm64
          os: linux
```

Another option is to use the `aliases/prefixes` property not only to specify the aliases for a given plugin but also the plugin name too. As a result, we can avoid conflicts when the same name is used for a different executors. This can be added later, once requested by end-users.

</details>
