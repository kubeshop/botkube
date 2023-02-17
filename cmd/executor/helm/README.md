# Helm executor

Helm is the Botkube executor plugin that allows you to run the Helm CLI commands directly from any communication platform.

## Manual testing

This section describes a manual testing of all supported Helm executor commands. It utilizes the `k3d` cluster and the leverages a built-in functionality to access your host system by referring to it as [`host.k3d.internal`](https://k3d.io/v5.0.1/faq/faq/#how-to-access-services-like-a-database-running-on-my-docker-host-machine).

Only Slack integration is described, but described steps are the same for all communication platforms.

### Install Botkube

1. Export Slack tokens:
   ```bash
   export SLACK_API_APP_TOKEN={token} # should start with xapp-1-
   export SLACK_API_BOT_TOKEN={token} # should start with xoxb-
   ```
1. Create `/tmp/test-values.yaml`:

   ```bash
   cat << EOF > /tmp/test-values.yaml
   communications:
     default-group:
       socketSlack:
         enabled: true
         channels:
           default:
             name: random
             bindings:
               executors:
                 - 'plugin-based'
         appToken: "${SLACK_API_APP_TOKEN}"
         botToken: "${SLACK_API_BOT_TOKEN}"

   executors:
     'plugin-based':
       botkube/helm:
         enabled: true

   extraEnv:
     - name: LOG_LEVEL_EXECUTOR_BOTKUBE_HELM
       value: "debug"

   plugins:
     cacheDir: "/tmp/plugins"
     repositories:
       botkube:
         url: http://host.k3d.internal:3000/botkube.yaml

   rbac:
     create: true
     rules:
       - apiGroups: ["*"]
         resources: ["*"]
         verbs: ["get", "watch", "list", "create", "delete", "update", "patch"]
   EOF
   ```

1. Start `k3d` cluster

   ```bash
   k3d cluster create labs --image=rancher/k3s:v1.25.0-k3s1
   ```

1. Build plugins:

   ```bash
   make build-plugins
   ```

1. Start plugin server:

   ```bash
   env PLUGIN_SERVER_HOST=http://host.k3d.internal go run test/helpers/plugin_server.go
   ```

1. Install Botkube

   ```bash
   helm upgrade botkube --install --namespace botkube ./helm/botkube --wait --create-namespace \
     -f /tmp/test-values.yaml
   ```

### Steps

1. Global Help

   ```bash
   @Botkube helm help
   ```

1. Version

   ```bash
   @Botkube helm version -h
   @Botkube helm version
   ```

1. Install

   ```bash
   @Botkube helm install -h

   # By absolute URL:
   @Botkube helm install
     --repo https://charts.bitnami.com/bitnami psql postgresql
     --set clusterDomain='testing.local'

   # By chart reference:
   @Botkube helm install https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --create-namespace -n test --generate-name
   ```

1. List

   ```bash
   @Botkube helm list -h
   @Botkube helm list -A
   @Botkube helm list -f 'p' -A
   ```

1. Status

   ```bash
   @Botkube helm status -h
   @Botkube helm status psql
   ```

1. Upgrade

   ```bash
   @Botkube helm upgrade -h
   @Botkube helm upgrade --repo https://charts.bitnami.com/bitnami psql postgresql --set clusterDomain='cluster.local'
   ```

1. History

   ```bash
   @Botkube helm history -h
   @Botkube helm history psql
   @Botkube helm hist psql
   @Botkube helm history psql -o json
   ```

1. Get

   ```bash
   @Botkube helm get -h
   ```

1. Get all

   ```bash
   @Botkube helm get all -h
   @Botkube helm get all psql
   @Botkube helm get all psql --template {{.Release.Name}}
   ```

1. Get hooks

   ```bash
   @Botkube helm get hooks -h
   @Botkube helm get hooks psql
   @Botkube helm get hooks psql --revision 1
   @Botkube helm get hooks psql --revision 2
   @Botkube helm get hooks psql --revision 3
   ```

1. Get manifest

   ```bash
   @Botkube helm get manifest -h
   @Botkube helm get manifest psql
   ```

1. Get notes

   ```bash
   @Botkube helm get notes -h
   @Botkube helm get notes psql -n test2
   ```

1. Get values

   ```bash
   @Botkube helm get values -h
   @Botkube helm get values psql
   @Botkube helm get values psql --all
   @Botkube helm get values psql --all --output json --revision 1
   ```

1. Rollback

   ```bash
   @Botkube helm rollback -h
   @Botkube helm rollback psql
   ```

1. Test

   ```bash
   @Botkube helm test -h
   @Botkube helm test psql
   ```

1. Uninstall

   ```bash
   @Botkube helm uninstall -h
   @Botkube helm uninstall psql
   ```

1. Unknown flag

   ```bash
   @Botkube helm uninstall psql --random-flag
   ```

1. Known but not supported flag

   ```bash
   @Botkube helm install --repo https://charts.bitnami.com/bitnami psql postgresql --wait
   ```
