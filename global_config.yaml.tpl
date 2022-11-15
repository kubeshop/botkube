# Format: actions.{alias}
actions:
  # kubectl based action.
  'show-created-resource':
    # If true, enables the action.
    enabled: false

    # Action display name posted in the channels bound to the same source bindings.
    displayName: "Display created resource"
    # A text value denoting the command run by this action, may contain even based templated values.
    # The executor is inferred directly from the command, e.g. here we require a kubectl executor
    command: "kubectl describe {{ .Event.TypeMeta.Kind | lower }}{{ if .Event.Namespace }} -n {{ .Event.Namespace }}{{ end }} {{ .Event.Name }}"

    # Bindings for a given action.
    bindings:
      # Sources of events that trigger a given action.
      sources:
        - k8s-create-events
      # Executors configuration for a given automation.
      executors:
        - kubectl-read-only
  'show-logs-on-error':
    # If true, enables the action.
    enabled: false

    # Action display name posted in the channels bound to the same source bindings.
    displayName: "Show logs on error"
    # A text value denoting the command run by this action, may contain even based templated values.
    # The executor is inferred directly from the command, e.g. here we require a kubectl executor
    command: "kubectl logs {{ .Event.TypeMeta.Kind | lower }}/{{ .Event.Name }} -n {{ .Event.Namespace }}"

    # Bindings for a given action.
    bindings:
      # Sources of events that trigger a given action.
      sources:
        - k8s-err-with-logs-events
      # Executors configuration for a given automation.
      executors:
        - kubectl-read-only

# Map of sources. Source contains configuration for Kubernetes events and sending recommendations.
# The property name under `sources` object is an alias for a given configuration. You can define multiple sources configuration with different names.
# Key name is used as a binding reference.
#
## Format: sources.{alias}
sources:
  'k8s-recommendation-events':
    displayName: "Kubernetes Recommendations"
    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes configuration for various recommendation insights.
      recommendations:
        # Recommendations for Pod Kubernetes resource.
        pod:
          # If true, notifies about Pod containers that use `latest` tag for images.
          noLatestImageTag: true
          # If true, notifies about Pod resources created without labels.
          labelsSet: true
        # Recommendations for Ingress Kubernetes resource.
        ingress:
          # If true, notifies about Ingress resources with invalid backend service reference.
          backendServiceValid: true
          # If true, notifies about Ingress resources with invalid TLS secret reference.
          tlsSecretValid: true

  'k8s-all-events':
    displayName: "Kubernetes Info"
    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: &k8s-events-namespaces
        # Include contains a list of allowed Namespaces.
        # It can also contain a regex expressions:
        #  `- ".*"` - to specify all Namespaces.
        include:
          - ".*"
        # Exclude contains a list of Namespaces to be ignored even if allowed by Include.
        # It can also contain a regex expressions:
        #  `- "test-.*"` - to specif all Namespaces with `test-` prefix.
        # exclude: []

      # Describes event constraints for Kubernetes resources.
      # These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object.
      event:
        # Lists all event types to be watched.
        types:
          - create
          - delete
          - error

      # Describes the Kubernetes resources to watch.
      # Resources are identified by its type in `{group}/{version}/{kind (plural)}` format. Examples: `apps/v1/deployments`, `v1/pods`.
      # Each resource can override the namespaces and event configuration by using dedicated `event` and `namespaces` field.
      resources:
        - type: v1/pods
        #  namespaces:             # Overrides 'source'.kubernetes.namespaces
        #    include:
        #      - ".*"
        #    exclude: []
        - type: v1/services
        - type: networking.k8s.io/v1/ingresses
        - type: v1/nodes
        - type: v1/namespaces
        - type: v1/persistentvolumes
        - type: v1/persistentvolumeclaims
        - type: v1/configmaps
        - type: rbac.authorization.k8s.io/v1/roles
        - type: rbac.authorization.k8s.io/v1/rolebindings
        - type: rbac.authorization.k8s.io/v1/clusterrolebindings
        - type: rbac.authorization.k8s.io/v1/clusterroles
        - type: apps/v1/daemonsets
          event: # Overrides 'source'.kubernetes.event
            types:
              - create
              - update
              - delete
              - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.numberReady
        - type: batch/v1/jobs
          event: # Overrides 'source'.kubernetes.event
            types:
              - create
              - update
              - delete
              - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.conditions[*].type
        - type: apps/v1/deployments
          event: # Overrides 'source'.kubernetes.event
            types:
              - create
              - update
              - delete
              - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.availableReplicas
        - type: apps/v1/statefulsets
          event: # Overrides 'source'.kubernetes.event
            types:
              - create
              - update
              - delete
              - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.readyReplicas
       ## Custom resource example
       # - type: velero.io/v1/backups
       #   namespaces:
       #     include:
       #       - ".*"
       #     exclude:
       #       -
       #   event:
       #     types:
       #       - create
       #       - update
       #       - delete
       #       - error
       #   updateSetting:
       #     includeDiff: true
       #     fields:
       #       - status.phase

  'k8s-err-events':
    displayName: "Kubernetes Errors"

    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: *k8s-events-namespaces

      # Describes event constraints for Kubernetes resources.
      # These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object.
      event:
        # Lists all event types to be watched.
        types:
          - error

      # Describes the Kubernetes resources you want to watch.
      resources:
        - type: v1/pods
        - type: v1/services
        - type: networking.k8s.io/v1/ingresses
        - type: v1/nodes
        - type: v1/namespaces
        - type: v1/persistentvolumes
        - type: v1/persistentvolumeclaims
        - type: v1/configmaps
        - type: rbac.authorization.k8s.io/v1/roles
        - type: rbac.authorization.k8s.io/v1/rolebindings
        - type: rbac.authorization.k8s.io/v1/clusterrolebindings
        - type: rbac.authorization.k8s.io/v1/clusterroles
        - type: apps/v1/deployments
        - type: apps/v1/statefulsets
        - type: apps/v1/daemonsets
        - type: batch/v1/jobs
  'k8s-err-with-logs-events':
    displayName: "Kubernetes Errors for resources with logs"

    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: *k8s-events-namespaces

      # Describes event constraints for Kubernetes resources.
      # These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object.
      event:
        # Lists all event types to be watched.
        types:
          - error

      # Describes the Kubernetes resources you want to watch.
      resources:
        - type: v1/pods
        - type: apps/v1/deployments
        - type: apps/v1/statefulsets
        - type: apps/v1/daemonsets
        - type: batch/v1/jobs
        # `apps/v1/replicasets` excluded on purpose - to not show logs twice for a given higher-level resource (e.g. Deployment)

  'k8s-create-events':
    displayName: "Kubernetes Resource Created Events"

    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: *k8s-events-namespaces

      # Describes event constraints for Kubernetes resources.
      # These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object.
      event:
        # Lists all event types to be watched.
        types:
          - create

      # Describes the Kubernetes resources you want to watch.
      resources:
        - type: v1/pods
        - type: v1/services
        - type: networking.k8s.io/v1/ingresses
        - type: v1/nodes
        - type: v1/namespaces
        - type: v1/configmaps
        - type: apps/v1/deployments
        - type: apps/v1/statefulsets
        - type: apps/v1/daemonsets
        - type: batch/v1/jobs

# Filter settings for various sources.
# Currently, all filters are globally enabled or disabled.
# You can enable or disable filters with `@Botkube filters` commands.
filters:
  kubernetes:
    # If true, enables support for `botkube.io/disable` and `botkube.io/channel` resource annotations.
    objectAnnotationChecker: true
    # If true, filters out Node-related events that are not important.
    nodeEventsChecker: true

# Setting to support multiple clusters
settings:
  # Cluster name to differentiate incoming messages
  clusterName: not-configured
  # Set true to enable config watcher
  # Server configuration which exposes functionality related to the app lifecycle.
  lifecycleServer:
    deployment:
      name: botkube
      namespace: botkube
    port: "2113"
  # Set false to disable upgrade notification
  upgradeNotifier: true

# Parameters for the config watcher container.
configWatcher:
    enabled: false # Used only on Kubernetes

# Map of enabled executors. The `executors` property name is an alias for a given configuration.
# It's used as a binding reference.
#
# Format: executors.{alias}
executors:
  'kubectl-read-only':
    # Kubectl executor configs
    kubectl:
      namespaces:
        include: [".*"]
      # Set true to enable kubectl commands execution
      enabled: false
      # List of allowed commands
      commands:
        # method which are allowed
        verbs: [ "api-resources", "api-versions", "cluster-info", "describe", "diff", "explain", "get", "logs", "top", "auth" ]
        # resource configuration which is allowed
        resources: [ "deployments", "pods" , "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes" ]
      # set Namespace to execute botkube kubectl commands by default
      defaultNamespace: default
      # Set true to enable commands execution from configured channel only
      restrictAccess: false
