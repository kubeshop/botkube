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
    command: "kubectl describe {{ .Event.TypeMeta.Kind | lower }} {{ if .Event.Namespace -}}-n {{ .Event.Namespace }}{{- end }} {{ .Event.Name }}"

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

# Map of enabled sources. The `source` property name is an alias for a given configuration.
# It's used as a binding reference.
#
# Format: source.{alias}
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

      # Describes events for every Kubernetes resources you want to watch or exclude.
      # These events are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own events object.
      events:
        - create
        - delete
        - error

      # Describes the Kubernetes resources you want to watch.
      resources:
        - name: v1/pods             # Name of the resource. Resource name must be in group/version/resource (G/V/R) format
                                    # resource name should be plural (e.g apps/v1/deployments, v1/pods)

        #  namespaces:             # Overrides 'source'.kubernetes.namespaces
        #    include:
        #      - ".*"
        #    exclude: []
        - name: v1/services
        - name: networking.k8s.io/v1/ingresses
        - name: v1/nodes
        - name: v1/namespaces
        - name: v1/persistentvolumes
        - name: v1/persistentvolumeclaims
        - name: v1/configmaps
        - name: rbac.authorization.k8s.io/v1/roles
        - name: rbac.authorization.k8s.io/v1/rolebindings
        - name: rbac.authorization.k8s.io/v1/clusterrolebindings
        - name: rbac.authorization.k8s.io/v1/clusterroles
        - name: apps/v1/daemonsets
          events: # Overrides 'source'.kubernetes.events
            - create
            - update
            - delete
            - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.numberReady
        - name: batch/v1/jobs
          events: # Overrides 'source'.kubernetes.events
            - create
            - update
            - delete
            - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.conditions[*].type
        - name: apps/v1/deployments
          events: # Overrides 'source'.kubernetes.events
            - create
            - update
            - delete
            - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.availableReplicas
        - name: apps/v1/statefulsets
          events: # Overrides 'source'.kubernetes.events
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
       # - name: velero.io/v1/backups
       #   namespaces:
       #     include:
       #       - ".*"
       #     exclude:
       #       -
       #   events:
       #     - create
       #     - update
       #     - delete
       #     - error
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

      # Describes events for every Kubernetes resources you want to watch or exclude.
      # These events are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own events object.
      events:
        - error

      # Describes the Kubernetes resources you want to watch.
      resources:
        - name: v1/pods
        - name: v1/services
        - name: networking.k8s.io/v1/ingresses
        - name: v1/nodes
        - name: v1/namespaces
        - name: v1/persistentvolumes
        - name: v1/persistentvolumeclaims
        - name: v1/configmaps
        - name: rbac.authorization.k8s.io/v1/roles
        - name: rbac.authorization.k8s.io/v1/rolebindings
        - name: rbac.authorization.k8s.io/v1/clusterrolebindings
        - name: rbac.authorization.k8s.io/v1/clusterroles
        - name: apps/v1/deployments
        - name: apps/v1/statefulsets
        - name: apps/v1/daemonsets
        - name: batch/v1/jobs
  'k8s-err-with-logs-events':
    displayName: "Kubernetes Errors for resources with logs"

    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: *k8s-events-namespaces

      # Describes events for every Kubernetes resources you want to watch or exclude.
      # These events are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own events object.
      events:
        - error

      # Describes the Kubernetes resources you want to watch.
      resources:
        - name: v1/pods
        - name: apps/v1/deployments
        - name: apps/v1/statefulsets
        - name: apps/v1/daemonsets
        - name: batch/v1/jobs
        # replicasets excluded on purpose - to not show logs twice for a given e.g. deployment + replicaset

  'k8s-create-events':
    displayName: "Kubernetes Resource Created Events"

    # Describes Kubernetes source configuration.
    kubernetes:
      # Describes namespaces for every Kubernetes resources you want to watch or exclude.
      # These namespaces are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own namespaces object.
      namespaces: *k8s-events-namespaces

      # Describes events for every Kubernetes resources you want to watch or exclude.
      # These events are applied to every resource specified in the resources list.
      # However, every specified resource can override this by using its own events object.
      events:
        - create

      # Describes the Kubernetes resources you want to watch.
      resources:
        - name: v1/pods
        - name: v1/services
        - name: networking.k8s.io/v1/ingresses
        - name: v1/nodes
        - name: v1/namespaces
        - name: v1/persistentvolumes
        - name: v1/persistentvolumeclaims
        - name: v1/configmaps
        - name: rbac.authorization.k8s.io/v1/roles
        - name: rbac.authorization.k8s.io/v1/rolebindings
        - name: rbac.authorization.k8s.io/v1/clusterrolebindings
        - name: rbac.authorization.k8s.io/v1/clusterroles
        - name: apps/v1/deployments
        - name: apps/v1/statefulsets
        - name: apps/v1/daemonsets
        - name: batch/v1/jobs

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
