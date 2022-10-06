# Map of enabled sources. The `source` property name is an alias for a given configuration.
# It's used as a binding reference.
#
# Format: source.<alias>
sources:
  'k8s-events':

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

      # TODO: https://github.com/kubeshop/botkube/issues/596
      # New 'namespace' property.
      # It can be overridden in the nested level.
      # namespace:
      #   include: [ ".*" ]
      resources:
        - name: v1/pods             # Name of the resource. Resource name must be in group/version/resource (G/V/R) format
                                    # resource name should be plural (e.g apps/v1/deployments, v1/pods)
          namespaces:
            # Include contains a list of allowed Namespaces.
            # It can also contain a regex expressions:
            #  - ".*" - to specify all Namespaces.
            include:
              - ".*"
            # Exclude contains a list of Namespaces to be ignored even if allowed by Include.
            # It can also contain a regex expressions:
            #  - "test-.*" - to specif all Namespaces with `test-` prefix.
            #exclude: []
          events:                   # List of lifecycle events you want to receive, e.g create, update, delete, error OR all
            - create
            - delete
            - error
        - name: v1/services
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: apps/v1/deployments
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
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
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - update
            - delete
            - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.readyReplicas
        - name: networking.k8s.io/v1/ingresses
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/nodes
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/namespaces
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/persistentvolumes
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/persistentvolumeclaims
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/secrets
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: v1/configmaps
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: apps/v1/daemonsets
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
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
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - update
            - delete
            - error
          updateSetting:
            includeDiff: true
            fields:
              - spec.template.spec.containers[*].image
              - status.conditions[*].type
        - name: rbac.authorization.k8s.io/v1/roles
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: rbac.authorization.k8s.io/v1/rolebindings
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - delete
            - error
        - name: rbac.authorization.k8s.io/v1/clusterrolebindings
          namespaces:
            include:
              - ".*"
            exclude:
              -
          events:
            - create
            - error
        - name: v1/services
          events:
            - create
            - delete
            - error

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
# Format: executors.<alias>
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
