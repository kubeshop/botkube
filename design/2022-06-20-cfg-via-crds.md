# Configure BotKube via CRs

Created on 2022-06-14 by Mateusz Szostok ([@mszostok](https://github.com/mszostok))

## Motivation

See [2022-06-15-api-problems.md](2022-06-15-api-problems.md).

## Overview

> Make BotKube, Kube native.

Defining configuration via CRD allows:

- Easy extensibility - new automation that watches for updates on the new object.
  - Currently, one big YAML file, requires BotKube restart, extensions need to be built-in.
- Showing status of a given extension - if it's up and running or there were some errors.
  - Now we can check that only in BotKube logs.
- Providing metadata information about given extension. Will be useful for discoverability.
  - Currently, not available.
- Configuration that will be cluster-wide and namespace-scoped.
  - Currently, it's possible but not in a native K8s way.
- Out-of-the-box validation via Open API schema.
  - Currently, not available.

However, switching from config files to CRDs also adds some limitations:

- Limit 1MB per configuration definition.
- Configuration is purly K8s based, so it means no option to run on bare-metal/docker/etc.
- Currently, we have two config YAMLs that can be configured. Namespace-scoped CR will make it harder. For example, you
  need to ensure that Secret with a communicator token is in all NS where communication CR was created. It has pros and
  cons. A pros are definitely a better security model and fine-grained approach.

  > **REMEDIATION:** To simplify transition, we can start with the cluster-wide CR, that will define globally settings - same as current YAML files. Later we can introduce namespaced version to allow fine-grained configuration.

### Design

Domains:

1. Communicators
    1. (Cluster)CommunicatorTemplate
    2. (Cluster)Slack/(Cluster)Discord/(Cluster)MSTeams/etc.
2. Mutators (filters)
    1. (Cluster)MutatorTemplate
    2. (Cluster)Mutator

    Currently, I don't see any candidate for this.

    | Filter Name             | DESCRIPTION                                                                       | Note                                 |
    |-------------------------|-----------------------------------------------------------------------------------|--------------------------------------|
    | ImageTagChecker         | Checks and adds recommendation if 'latest' image tag is used for container image. | Move as  notificator.                |
    | IngressValidator        | Checks if services and tls secrets used in ingress specs are available.           | Move as notificator.                 |
    | ObjectAnnotationChecker | Checks if annotations botkube.io/* present in object specs and filters them.      | Move as notificator.                 |
    | PodLabelChecker         | Checks and adds recommendations if labels are missing in the pod specs.           | Move as notificator.                 |
    | NamespaceChecker        | Checks if event belongs to blocklisted namespaces and filter them.                | Remove. It will be per resource now. |
    | NodeEventsChecker       | Sends notifications on node level critical events.                                | Move as notificator.                 |

3. Executors
    1. (Cluster)ExecutorTemplate
    2. (Cluster)Executor
4. Notificators (including validators)
    1. (Cluster)NotificationTemplate
    2. (Cluster)Notification

Initially, all executors and notificators can be marked as built-in. The `spec.plugin.built-in: true` marks that a given functionality is built-in. We can later extract it into separate plugin (probably Docker image).

#### Communicator

Communicators integration in BotKube is quite narrow in comparison to executors or notificators, and it's not the main extension part. We can even decide to represent as fixed CRDs, see [Communicators CRDs](#communicator-crds). In the first implementation also the Namespace-scoped CRD doesn't make sens.

```yaml
apiVersion: "core.botkube.io/v1"
kind: ClusterCommunicatorTemplate
metadata:
  name: Slack
spec:
  plugin:
    built-in: slack
  metadata:
    displayName: Slack communicator
    description: Connector for Slack communicator that helps to monitor your Kubernetes cluster, debug deployments and run custom checks on resource specs.
    license:
      name: "MIT"
    documentationURL: https://examples.com/docs
    supportURL: https://example.com/online-support
    iconURL: https://examples.com/favicon.ico
    maintainers:
      - email: foo@example.com
        name: Foo Bar
        url: https://examples.com/foo/bar
status:
  phase: Registered/Failed
  message: "CRD 'clusterslack.communicators.core.botkube.io/v1' not registered in cluster."
---
apiVersion: "core.botkube.io/v1"
kind: ClusterCommunicator
metadata:
  name: slack-instance
spec:
  template: Slack
  parameters:
    notiftype: short
    token:
      # value: <plain_data>
      valueFrom:
        secretKeyRef:
          name: communication-slack
      namespace: botkube-system
        key: token
    channels:
      - name: nodes # it's better with string as YAML doesn't support #, or @ chars
        bindings:
          notifications:
            - name: nodes-errors
          executors:
            - name: nodes-readonly
status:
  phase: Initializing/Connected/Failed
  message: "connection failed: 401 Unautorized"
  # Reflects the generation of the most recently observed change.
  #observedGeneration:
  # Last time the condition transitioned from one status to another.
  #lastTransitionTime:
```

#### Notifications

```yaml
apiVersion: "core.botkube.io/v1"
kind: ClusterNotificationTemplate
metadata:
  name: Kubernetes
spec:
  # instead of CRD, define schema, we can even skip when 'plugin.built-in: {name}`
  validation:
    # Schema for the `parameters` field
    openAPIV3Schema:
      properties:
        labels:
          type: array
          items: string

  plugin:
    built-in: k8s # needed for the first phase, when we don't want to extract all logic into separate plugins/Docker images.
    # later:
    #image: ghcr.io/kubeshop/botkube/k8s-notificator:v0.1.0

  metadata:
    displayName: Notify about Kuberentes events.
    license:
      name: "MIT"
    documentationURL: https://examples.com/docs
    supportURL: https://example.com/online-support
    iconURL: https://examples.com/favicon.ico
    maintainers:
      - email: foo@example.com
        name: Foo Bar
        url: https://examples.com/foo/bar
status:
  phase: Registered/Failed
---
apiVersion: "core.botkube.io/v1"
kind: ClusterNotification
metadata:
  name: k8s-network-errors
spec:
  template: Kubernetes # name of ClusterNotificationTemplate
  parameters: # Core webhook for ClusterNotification, validates it against 'ClusterNotificationTemplate[Kubernetes].validation.openAPIV3Schema'
    namespaces:  # global, can be OVERRIDDEN per resource
      include:
        - all
      resources:
        - name: v1/pods
          namespaces: # it overrides the top one!
            include:
              - istio-system
          events:
            - error
        - name: v1/services
          events:
            - error
        - name: networking.istio.io/v1alpha3/DestinationRules
          events:
            - error
        - name: networking.istio.io/v1alpha3/VirtualServices
          events:
            - error
status:
  phase: Initializing/Serving/Failed
---
apiVersion: "core.botkube.io/v1"
kind: Notification
metadata:
  name: k8s-network-errors
  namespace: team-a # only here Namespace is allowed. All events are scoped to this one.
spec:
  template: Kubernetes # refers Notification not ClusterNotification
  parameters:
    resources:
      - name: v1/pods
        events:
          - error
      - name: v1/services
        events:
          - error
      - name: networking.istio.io/v1alpha3/DestinationRules
        events:
          - error
      - name: networking.istio.io/v1alpha3/VirtualServices
        events:
          - error
```

#### Executors

```yaml
apiVersion: "core.botkube.io/v1"
kind: ClusterExecutorTemplate
metadata:
  name: Kubectl
spec:
  # instead of CRD, define schema:
  validation:
    # Schema for the `parameters` field
    openAPIV3Schema:
      properties:
        labels:
          type: array
          items: string
  plugin:
    built-in: kubectl # built-in name, so we know which one to pick. we can also use 'metadata.name'.
  metadata:
    displayName: Execute Kubectl CLI
    license:
      name: "MIT"
    documentationURL: https://examples.com/docs
    supportURL: https://example.com/online-support
    iconURL: https://examples.com/favicon.ico
    maintainers:
      - email: foo@example.com
        name: Foo Bar
        url: https://examples.com/foo/bar

status:
  phase: Registered/Failed
---
apiVersion: "core.botkube.io/v1"
kind: ClusterExecutor
metadata:
  name: kubectl-readonly
spec:
  template: Kubectl
  parameters:
    namespaces:
      include:
        - all
    commands:
      # method which are allowed
      verbs: ["get", "logs"]
      # resource configuration which is allowed
      resources: ["Deployments", "Pods", "Services"]
---
apiVersion: "core.botkube.io/v1"
kind: Executor
metadata:
  name: kubectl-readonly
  namespace: team-a
spec:
  template: Kubectl
  parameters:
    commands:
      # method which are allowed
      verbs: ["get", "logs"]
      # resource configuration which is allowed
      resources: ["Deployments", "Pods", "Services"]
```

### TODO

Show the target design and phases:
1. Add new "API"
   1. Stay with two files and later map it to CRDs?
   2. or go with CRDs on a feature flag
2. Only represent Built-in and allow configuration
3. Extract them as plugins (docker images)
   1. As a BotKube we don't care?
4. Then GraphQL etc.
   1. GraphQL service that shows the catalog of possible "sinks", "communicators", "executors"
  This will also enable predefined some catalog of policies:


# Archive

## Ideas

Install of it results in a CR creation and also registering a dedicated CRD

## Domains

1. Communicators
2. Filters
3. Mutators
4. Validators
5. Executors
6. Notificators

## Individual CRD approach

Each communicator, executor, and notificator is represented by own CRD.

### Communicator CRDs

Communicators integration in BotKube is quite narrow in comparison to executors or notificators, and it's not the main extension part.
To simplify the BotKube implementation we can still have them as built-in. To still be K8s native, we can define them as CRDs. For example:

- `Slack/ClusterSlack.communicators.core.botkube.io/v1`
- `Discord/ClusterDiscord.communicators.core.botkube.io/v1`
- `Mattermost/ClusterMattermost.communicators.core.botkube.io/v1`
- `MSTeams/ClusterMSTeams.communicators.core.botkube.io/v1`
- `Elasticsearch/ClusterElasticsearch.communicators.core.botkube.io/v1`

```yaml
apiVersion: "core.botkube.io/v1"
kind: (Cluster)CommunicatorTemplate
metadata:
  name: slack-instance
spec:
  crd:
    name: ClusterSlack.communicators.core.botkube.io/v1
  metadata:
    name: Slack
    displayName: Slack communicator
    description: Connector for Slack communicator that helps to monitor your Kubernetes cluster, debug deployments and run custom checks on resource specs.
    license:
      name: "MIT"
    documentationURL: https://examples.com/docs
    supportURL: https://example.com/online-support
    iconURL: https://examples.com/favicon.ico
    maintainers:
      - email: foo@example.com
        name: Foo Bar
        url: https://examples.com/foo/bar
status:
  phase: Registered/Failed
  message: "CRD 'clusterslack.communicators.core.botkube.io/v1' not registered in cluster."
---
apiVersion: "communicators.core.botkube.io/v1"
kind: ClusterSlack
metadata:
  name: slack-instance
spec:
  notiftype: short
  token:
    # value: <plain_data>
    valueFrom:
      secretKeyRef:
        name: communication-slack
    namespace: botkube-system
      key: token
  channels:
    - name: nodes # it's better with string as YAML doesn't support #, or @ chars
      bindings:
        notifications:
          - kind: ClusterKubernetes
            name: nodes-errors
        executors:
          - kind: ClusterKubectl
            name: nodes-readonly
status:
  phase: Initializing/Connected/Failed
  message: "connection failed: 401 Unautorized"
  # Reflects the generation of the most recently observed change.
  #observedGeneration:
  # Last time the condition transitioned from one status to another.
  #lastTransitionTime:
```


### Kubectl Executor CRD

```yaml
apiVersion: "executors.core.botkube.io/v1"
kind: ClusterKubectl
metadata:
  name: kubectl-readonly
spec:
  namespaces:
    include:
      - all
  commands:
    # method which are allowed
    verbs: ["get", "logs"]
    # resource configuration which is allowed
    resources: ["Deployments", "Pods", "Services"]
---
apiVersion: "executors.core.botkube.io/v1"
kind: Kubectl
metadata:
  name: kubectl-readonly
  namespace: team-a
spec:
  commands:
    # method which are allowed
    verbs: ["get", "logs"]
    # resource configuration which is allowed
    resources: ["Deployments", "Pods", "Services"]
```

### K8s notification CRD

```yaml
apiVersion: "notifications.core.botkube.io/v1"
kind: Notifications
metadata:
  name: network-errors
spec:
  namespaces:  # global, can be OVERRIDDEN per resource
    include:
      - all

  resources:
    - name: v1/pods
      namespaces: # it overrides the top one!
        include:
          - istio-system
      events:
        - error
    - name: v1/services
      events:
        - error
    - name: networking.istio.io/v1alpha3/DestinationRules
      events:
        - error
    - name: networking.istio.io/v1alpha3/VirtualServices
      events:
        - error
---
apiVersion: "notifications.core.botkube.io/v1"
kind: Kubernetes
metadata:
  name: network-errors
  namespace: team-a # only here Namespace is allowed. All events are scoped to this one.
spec:
  resources:
    - name: v1/pods
      events:
        - error
    - name: v1/services
      events:
        - error
    - name: networking.istio.io/v1alpha3/DestinationRules
      events:
        - error
    - name: networking.istio.io/v1alpha3/VirtualServices
      events:
        - error
```


