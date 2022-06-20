### Communications

The communications settings are stored in the `comm_config.yaml` file.

Issues:
1. One huge YAML that you need to scroll.
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
1. Split into messaging programs and "sinks"?

### Resource configuration

The resource settings are stored in the `resource_config.yaml` file.

The resource configuration file contains:
- Notification settings about K8s events
- Kubectl command executor settings
- global settings like cluster name, upgrade notifications settings, etc.
- Information if recommendations should be enabled


Issues:
- One huge YAML that you need to scroll.
- it holds too much different configuration
- You are not able to enable/disable a given recommendation: `recommendations: true`
- `kubectl` is under `settings`
- it will be good to define it multiple times. Now you need to deploy botkube twice if you want to have two different notification strategies.


Ideas:
1. Extract `settings.kubectl` to `executors[].kubectl`. In the future we will add there more.
2. Nest the `resources` under `notifications[].kubernetes.events`

```yaml
## test_config.yaml for Integration Testing
## Resources you want to watch
resources:
  - name: v1/pods             # Name of the resource. Resource name must be in group/version/resource (G/V/R) format
                              # resource name should be plural (e.g apps/v1/deployments, v1/pods)
    namespaces:               # List of namespaces, "all" will watch all the namespaces
      include:
        - all
      ignore:                 # List of namespaces to be ignored (omitempty), used only with include: all, can contain a wildcard (*)
        -                     # example : include [all], ignore [x,y,secret-ns-*]
    events:                   # List of lifecycle events you want to receive, e.g create, update, delete, error OR all
      - create
      - delete
      - error
    updateSetting:
      includeDiff: true
      fields:
        - spec.template.spec.containers[*].image
        - status.availableReplicas

# Check true if you want to receive recommendations
# about the best practices for the created resource
recommendations: true

# Setting to support multiple clusters
settings:
  # Cluster name to differentiate incoming messages
  clustername: not-configured
  # Kubectl executor configs
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
  # Set true to enable config watcher
  configwatcher: true
  # Set false to disable upgrade notification
  upgradeNotifier: true
```

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
  prometheus:
    # ...

  # Check true if you want to receive recommendations
  # about the best practices for the created resource
  recommendations:
    - image    # "Checks and adds recommendation if 'latest' image tag is used for container image."
    - pod      # "Checks and adds recommendations if labels are missing in the pod specs."
    - ingress  # "Checks if services and tls secrets used in ingress specs are available."
```

2. Executors

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

BotKube settings:
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

It's cleaner, but we still need to be able to configure a given "notificator/executor" multiple times.

#### Named configurations

1. Notifications
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
```yaml
# preferred! we have more control on api
notifications:
  - name: foo-name # name used for bindings
    kubernetes: # maybe introduce own name?
      resources:
        - name: v1/nodes
          namespaces:
            include:
              - all
          events:
            - error
```

```yaml
executors:
  - name: kubectl-read-only # name used for bindings
    namespaces:
      include:
        - team-a
    kubectl:
      enabled: true
      # List of allowed commands
      commands:
        # method which are allowed
        verbs: ["api-resources", "api-versions", "cluster-info", "describe", "diff", "explain", "get", "logs", "top", "auth"]
        # resource configuration which is allowed
        resources: ["deployments", "pods" , "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes"]
		helm:
			enabled: true
```


#### Mapping with communicators

```yaml
communications: # having multiple slacks? or ES?
  - name: default
    slack:
      enabled: true
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

### Filters

Issues:
- You are not able to "disable/enable" them via config. It needs to be done via `@Botkube filters list/disable/enable` command.
- The filter package holds not only functionality to "filter" object but also to mutate or validate them. For example:
  - mutator: **Object Annotation Checker** - Checks if object has `"botkube.io/channel"` and if yes, change the default channel where notification will be sent.
  - validator: **IngressValidator** - "Checks if services and tls secrets used in ingress specs are available."
  - filter: **NamespaceChecker** - "Checks if event belongs to blocklisted namespaces and filter them."

Idea:
1. Separate them into `Filters`, `Mutators`, `Validators` - this can be to complex for now.
2. or rename `Filters` to `Mutators` and extract `Validators` under `Notificators`. Because `Validators` are mostly about recommendation, so belongs to `notificators`. Filters are almost the same as `Mutators`, so merge them under `Mutators` name.

### TODO

- unify naming? notifications vs executors? `notificator` + `executor`?
