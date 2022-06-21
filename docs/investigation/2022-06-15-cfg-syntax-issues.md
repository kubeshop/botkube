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

#### Extracted configuration

1. Notifications
    <table>
    <tr>
    <td> Before </td> <td> After </td>
    </tr>
    <tr>
    <td>

    ```yaml
    # Notify about K8s events
    resources:
      - name: v1/nodes
      namespaces:
        include:
          - all
      events:
        - error

    # Recommendations about the
    # best practices for the created resource
    recommendations: true
    ```

    </td>
    <td>

    ```yaml
    notifications:
      # Notify about K8s events
      kubernetes:
        resources:
          - name: v1/nodes
          namespaces:
            include:
              - all
          events:
            - error
      sysdig:
        # ..
      kubePug:
        # ...

      # Recommendations about the
      # best practices for the created resource
      recommendations:
        - image    # "Checks and adds recommendation if 'latest' image tag is used for container image."
        - pod      # "Checks and adds recommendations if labels are missing in the pod specs."
        - ingress  # "Checks if services and tls secrets used in ingress specs are available."
    ```

    </td>
    </tr>
    </table>


2. Executors
    <table>
    <tr>
    <td> Before </td> <td> After </td>
    </tr>
    <tr>
    <td>

    ```yaml
    settings:
      # Cluster name to differentiate incoming messages
      clustername: not-configured
      # Kubectl executor configs
      kubectl:
        enabled: false
        commands:
          verbs: ["api-resources", "...", "auth"]
          resources: ["deployments", "...", "nodes"]
    ```

    </td>
    <td>


    ```yaml
    executors:
      kubectl:
        enabled: false
        commands:
          verbs: ["api-resources", "...", "auth"]
          resources: ["deployments", "...", "nodes"]
      helm:
        # ...
      istioctl:
        # ...
    ```

    </td>
    </tr>
    </table>

3. BotKube settings:

    Stay as they are right now.

    ```yaml
    settings:
      # Cluster name to differentiate incoming messages
      clustername: not-configured
      # Set true to enable config watcher
      configwatcher: true
      # Set false to disable upgrade notification
      upgradeNotifier: true
    ```

The API is cleaner, but we still need to be able to configure a given "notificator/executor" multiple times. Let's introduce [named configuration](#named-configurations).

#### Named configurations

1. Notifications

    <table>
    <tr>
    <td> Before </td> <td> After </td>
    </tr>
    <tr>
    <td>

    ```yaml
    notifications:
      # Notify about K8s events
      kubernetes:
        resources:
          - name: v1/nodes
          namespaces:
            include:
              - all
          events:
            - error
      sysdig:
        # ..
      kubePug:
        # ...
    ```

    </td>
    <td>

    ```yaml
    notifications:
      - name: nodes-errors # name used for bindings
        kubernetes:
          resources:
            - name: v1/nodes
              namespaces:
                include:
                  - all
              events:
                - error
        sysdig:
          # ..
        kubePug:
          # ...
    ```

    </td>
    </tr>
    </table>

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

3. Executors

    <table>
    <tr>
    <td> Before </td> <td> After </td>
    </tr>
    <tr>
    <td>

    ```yaml
    executors:
      kubectl:
        enabled: false
        commands:
            verbs: ["api-resources", "...", "auth"]
            resources: ["deployments", "..", "nodes"]
      helm:
        # ...
      istioctl
        # ...
    ```

    </td>
    <td>

     ```yaml
    executors:
      - name: kubectl-read-only # name used for bindings
        kubectl:
          namespaces:
            include:
              - team-a
          commands:
            verbs: ["api-resources", "...", "auth"]
            resources: ["deployments", "..", "nodes"]
      - name: helm-full-access # name used for bindings
        helm:
          namespaces:
            include:
              - team-a
          commands:
            verbs: ["list", "delete", "install"]
    ```

    </td>
    </tr>
    </table>

## Mapping with communicators

Extend each "communication" platform with dedicated bindings:
```yaml
communications: # having multiple slacks? or ES?
  - name: tenant-b-workspace
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

Other option is to introduce "profiles/policies/presets" that can gather the given configuration together. See the [Polices](../proposal/2022-06-14-policies.md) proposal.

## Filters

### Issues

- You are not able to "disable/enable" them via config. It needs to be done via [`@Botkube filters list/disable/enable`](https://www.botkube.io/usage/#manage-filters) command.
- The filter package holds not only functionality to "filter" objects but also to mutate or validate them. For example:

    | Name                          | Type      | Description                                                                                                         |
    |-------------------------------|-----------|---------------------------------------------------------------------------------------------------------------------|
    | **Object Annotation Checker** | mutator   | Checks if object has `"botkube.io/channel"` and if yes, change the default channel where notification will be sent. |
    | **IngressValidator**          | validator | "Checks if services and tls secrets used in ingress specs are available."                                           |
    | **NamespaceChecker**          | filter    | "Checks if event belongs to blocklisted namespaces and filter them."                                                |

### Ideas

1. Separate them into `Filters`, `Mutators`, `Validators` - this can be too complex for now.
2. or rename `Filters` to `Mutators` and extract `Validators` under `Notificators`. Because `Validators` are mostly about recommendation, so belongs to `notificators`. Filters are almost the same as `Mutators`, so merge them under `Mutators` name.

   However, I don't see any candidate for `Mutators` right now.

   | Filter Name             | Description                                                                       | Note                                    |
   |-------------------------|-----------------------------------------------------------------------------------|-----------------------------------------|
   | ImageTagChecker         | Checks and adds recommendation if 'latest' image tag is used for container image. | Move as recommendation notificator.     |
   | IngressValidator        | Checks if services and tls secrets used in ingress specs are available.           | Move as recommendation notificator.     |
   | ObjectAnnotationChecker | Checks if annotations botkube.io/* present in object specs and filters them.      | Remove it.                              |
   | PodLabelChecker         | Checks and adds recommendations if labels are missing in the pod specs.           | Move as recommendation notificator.     |
   | NamespaceChecker        | Checks if event belongs to blocklisted namespaces and filter them.                | Remove it. It will be per resource now. |
   | NodeEventsChecker       | Sends notifications on node level critical events.                                | Move as K8s events notificator.         |

## Others

1. Unify naming between notifications vs executors. Maybe go with `notificator` and `executor`?
2. Get rid of the `all` name usage. Currently, user cannot have `all` as Namespace name however it can have `all` as a channel name. It's misleading in which scope it's a reserved name and in which not. It can be replaced with e.g. `@all`.

## Other issues

Issues that are still not addressed:
- Showing status of a given extension - if it's up and running.
  - Now we can check that only in BotKube logs.
- Providing metadata information about given extension (icon, display name, docs url etc.). Will be useful for discoverability.
  - Currently, not available.
- Out-of-the-box validation via Open API schema.
  - Currently, not available.
- Easy extensibility - add a new executor/notificator.
  - Currently, via built-in filters.

## Consequences

This section described necessary changes if proposal will be accepted.

### Minimum changes

1. The `resources` notifications are moved under `notifications[].kubernetes[].resources`.
2. Kubectl executor moved under `executors[].kubectl`.
3. The `namespaces.include` and `namespaces.exclude` properties are added to the `kubectl` executor.
3. The `namespaces.include` and `namespaces.exclude` properties are added to `notifications[].kubernetes[]`.
4. The `resource_config.yaml` and `comm_config.yaml` are merged into one, but you can provide config multiple times. In the same way, as Helm takes the `values.yaml` file. It's up to the user how it will be split.
5. Update documentation about configuration.

### Follow-up changes

1. Recommendations are merged under notifications.
2. Filters are removed and existing one are moved under `notifications[].recommandtions[]`.
3. Update `@BotKube` commands to reflect new configuration.
4. **Optional**: Add CLI to simplify creating/updating configuration.


