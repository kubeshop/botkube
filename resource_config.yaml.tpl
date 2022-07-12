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
  - name: v1/services
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: apps/v1/deployments
    namespaces:
      include:  
        - all
      ignore:
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
        - all
      ignore:
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
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/nodes
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/namespaces
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/persistentvolumes
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/persistentvolumeclaims
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/secrets
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: v1/configmaps
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: apps/v1/daemonsets
    namespaces:
      include:  
        - all
      ignore:
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
        - all
      ignore:
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
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: rbac.authorization.k8s.io/v1/rolebindings
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: rbac.authorization.k8s.io/v1/clusterrolebindings
    namespaces:
      include:  
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
  - name: rbac.authorization.k8s.io/v1/clusterroles
    namespaces:
      include:
        - all
      ignore:
        -
    events:
      - create
      - delete
      - error
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
