# Configuration API syntax issues

This document describes found issue with the current syntax for BotKube configuration.

## Communications

The communications settings are stored in the `comm_config.yaml` file.

Issues:
1. All communicator settings are in one YAML.
2. It contains communicators but also "sinks" like Elasticsearch.
   1. They are a bit different. The communicators are bidirectional while ES is only able to receive events.
   ```yaml
   # Channels configuration
   communications:
     # Settings for Slack
     slack:
       enabled: false
       channel: 'SLACK_CHANNEL'
       token: 'SLACK_API_TOKEN'

     # Settings for ELS
     elasticsearch:
       enabled: false
       server: 'ELASTICSEARCH_ADDRESS'           # e.g https://example.com:9243
       username: 'ELASTICSEARCH_USERNAME'        # Basic Auth
       # ELS index settings
       index:
         name: botkube
         type: botkube-event
         shards: 1
         replicas: 0
   ```

Ideas:
1. Split into messaging programs and "sinks"? Personally, I would do it later.

## Resource configuration

The resource settings are stored in the `resource_config.yaml` file.

The resource configuration file contains:
- Notification settings about K8s events
- Kubectl command executor settings
- global settings like cluster name, upgrade notifications settings, etc.
- Information if recommendations should be enabled


### Issues
- One huge YAML that you need to scroll.
- It holds too much different configuration.
- You are not able to enable/disable a given recommendation. There is only the `recommendations: true` property.
- The `kubectl` executor settings are under `settings`. It's not consistent with `resources` and `communications`.
- No option to define multiple notification settings. Currently, you need to deploy BotKube twice if you want to have two different notification strategies.

### Ideas

1. Extract `settings.kubectl` to `executors[].kubectl`. In the future we will add more executors there, e.g. `helm`, `istioctl` etc.
2. Nest `resources` under `notifications[].kubernetes.events`. In the future we will add more platforms that will send events, e.g. Sysdig, KubePug, etc.

Examples:

1. Notifications
    ```yaml
    notifications:
      kubernetes:
        resources:
          - name: v1/nodes
          namespaces:
            include:
              - all
          events:
            - error
      sysdig:
        # some settings..
      kubePug:
        # ...

      # Recommendations about the best practices for the created resource
      recommendations:
        - image    # "Checks and adds recommendation if 'latest' image tag is used for container image."
        - pod      # "Checks and adds recommendations if labels are missing in the pod specs."
        - ingress  # "Checks if services and tls secrets used in ingress specs are available."
    ```

3. Executors

    ```yaml
    executors:
      kubectl:
        # Set true to enable kubectl commands execution
        enabled: false
        # List of allowed commands
        commands:
          # method which are allowed
          verbs: ["api-resources", "api-versions", "cluster-info", "describe", "diff", "explain", "get", "logs", "top", "auth"]
          # resource configuration which is allowed
          resources: ["deployments", "pods" , "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes"]
        # set Namespace to execute botkube kubectl commands by default
        defaultNamespace: default
        # Set true to enable commands execution from configured channel only
        restrictAccess: false
    ```

4. BotKube settings:

    ```yaml
    # Setting to support multiple clusters
    settings:
      # Cluster name to differentiate incoming messages
      clustername: not-configured
      # Set true to enable config watcher
      configwatcher: true
      # Set false to disable upgrade notification
      upgradeNotifier: true
    ```

It's cleaner, but we still need to be able to configure a given "notificator/executor" multiple times. Let's introduce named configuration.

#### Named configurations

1. Notifications
    ```yaml
    notifications:
      - name: foo-name # name used for bindings
        kubernetes:
          resources:
            - name: v1/nodes
              namespaces:
                include:
                  - all
              events:
                - error
    ```

    <details>
      <summary>Discarded alternative</summary>

    ```yaml
    notifications:
      kubernetes:
        - name: nodes-errors # name used for bindings
          resources:
            - name: v1/nodes
            namespaces:
              include:
                - all
            events:
              - error
    ```

    </details>

2. Executors

    ```yaml
    executors:
      - name: kubectl-read-only # name used for bindings
        kubectl:
          namespaces:
            include:
              - team-a
          # List of allowed commands
          commands:
            # method which are allowed
            verbs: ["api-resources", "api-versions", "cluster-info", "describe", "diff", "explain", "get", "logs", "top", "auth"]
            # resource configuration which is allowed
            resources: ["deployments", "pods" , "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes"]
      - name: helm-full-access # name used for bindings
        helm:
          namespaces:
            include:
              - team-a
          commands:
            # method which are allowed
            verbs: ["list", "delete", "install"]
    ```


## Mapping with communicators

```yaml
communications: # having multiple slacks? or ES?
  - name: default
    slack:
      token: 'SLACK_API_TOKEN'
      # customized notifications
      channels:
        - name: "#nodes"
          bindings:
            notifications:
              - "nodes-errors"
              - "depreacted-api"
            executors:
              - "kubectl-read-only"
              - "helm-full-access"
```

## Filters

### Issues

- You are not able to "disable/enable" them via config. It needs to be done via `@Botkube filters list/disable/enable` command.
- The filter package holds not only functionality to "filter" object but also to mutate or validate them. For example:
  - mutator: **Object Annotation Checker** - Checks if object has `"botkube.io/channel"` and if yes, change the default channel where notification will be sent.
  - validator: **IngressValidator** - "Checks if services and tls secrets used in ingress specs are available."
  - filter: **NamespaceChecker** - "Checks if event belongs to blocklisted namespaces and filter them."

### Ideas

1. Separate them into `Filters`, `Mutators`, `Validators` - this can be too complex for now.
2. or rename `Filters` to `Mutators` and extract `Validators` under `Notificators`. Because `Validators` are mostly about recommendation, so belongs to `notificators`. Filters are almost the same as `Mutators`, so merge them under `Mutators` name.

   However, I don't see any candidate for `Mutators` right now.

   | Filter Name             | DESCRIPTION                                                                       | Note                                    |
   |-------------------------|-----------------------------------------------------------------------------------|-----------------------------------------|
   | ImageTagChecker         | Checks and adds recommendation if 'latest' image tag is used for container image. | Move as  notificator.                   |
   | IngressValidator        | Checks if services and tls secrets used in ingress specs are available.           | Move as notificator.                    |
   | ObjectAnnotationChecker | Checks if annotations botkube.io/* present in object specs and filters them.      | Remove it.                              |
   | PodLabelChecker         | Checks and adds recommendations if labels are missing in the pod specs.           | Move as notificator.                    |
   | NamespaceChecker        | Checks if event belongs to blocklisted namespaces and filter them.                | Remove it. It will be per resource now. |
   | NodeEventsChecker       | Sends notifications on node level critical events.                                | Move as notificator.                    |

## Others

1. Unify naming between notifications vs executors. Maybe go with `notificator` and `executor`?
2. Get rid of the `all` name usage. Currently, user cannot have `all` as Namespace name however it can have `all` as a channel name. It's misleading in which scope it's a reserved name and in which not. It can be replaced with e.g. `@all`.
