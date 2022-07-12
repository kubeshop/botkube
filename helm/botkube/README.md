# BotKube

![Version: v0.12.4](https://img.shields.io/badge/Version-v0.12.4-informational?style=flat-square) ![AppVersion: v0.12.4](https://img.shields.io/badge/AppVersion-v0.12.4-informational?style=flat-square)

Controller for the BotKube Slack app which helps you monitor your Kubernetes cluster, debug deployments and run specific checks on resources in the cluster.

**Homepage:** <https://botkube.io>

## Maintainers

| Name | Email  |
| ---- | ------ |
| BotKube Dev Team | <dev-team@botkube.io> |

## Source Code

* <https://github.com/kubeshop/botkube>

### Now Supports AWS IRSA on EKS

AWS has introduced IAM Role for Service Accounts in order to provide fine grained access. This is useful if you are looking to run BotKube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "<role_arn_to_assume>"
```

## Parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| [image.registry](./values.yaml#L4) | string | `"ghcr.io"` | BotKube container image registry. |
| [image.repository](./values.yaml#L6) | string | `"kubeshop/botkube"` | BotKube container image repository. |
| [image.pullPolicy](./values.yaml#L8) | string | `"IfNotPresent"` | BotKube container image pull policy. |
| [image.tag](./values.yaml#L10) | string | `"v9.99.9-dev"` | BotKube container image tag. Default tag is `appVersion` from Chart.yaml. |
| [podSecurityPolicy](./values.yaml#L14) | object | `{"enabled":false}` | Enable Pod Security Policy to allow BotKube to run in restricted clusters. [Ref doc](https://kubernetes.io/docs/concepts/policy/pod-security-policy/). |
| [securityContext](./values.yaml#L20) | object | Set to run as a Non-Privileged user. | Configure securityContext to manage user Privileges in Pod. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). |
| [containerSecurityContext](./values.yaml#L26) | object | `{"allowPrivilegeEscalation":false,"privileged":false,"readOnlyRootFilesystem":true}` | Configure Container Security Context. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). |
| [kubeconfig.enabled](./values.yaml#L34) | bool | `false` | If true, enables overriding the Kubernetes auth. |
| [kubeconfig.base64Config](./values.yaml#L36) | string | `""` | A base64 encoded kubeconfig that will be stored in a Secret, mounted to the Pod, and specified in the KUBECONFIG environment variable. |
| [kubeconfig.existingSecret](./values.yaml#L41) | string | `""` | A secret containing a kubeconfig to use.  |
| [log.level](./values.yaml#L46) | string | `"info"` | Set one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [log.disableColors](./values.yaml#L48) | bool | `false` | If true, disable ANSI colors in logging. |
| [config.resources](./values.yaml#L54) | list | "Watch all built-in K8s kinds" | Describe the Kubernetes resources you want to watch. |
| [config.recommendations](./values.yaml#L258) | bool | `true` | If true, BotKube sends recommendations about the best practices for the created resource. |
| [config.ssl.enabled](./values.yaml#L263) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [config.ssl.existingSecretName](./values.yaml#L269) | string | `""` | Using existing SSL secret. It MUST be in `botkube` Namespace.  |
| [config.ssl.cert](./values.yaml#L272) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [config.settings.clustername](./values.yaml#L277) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [config.settings.kubectl.enabled](./values.yaml#L281) | bool | `false` | If true, enables `kubectl` commands execution. |
| [config.settings.kubectl.commands.verbs](./values.yaml#L285) | list | `["api-resources","api-versions","cluster-info","describe","diff","explain","get","logs","top","auth"]` | Defines which `kubectl` methods are allowed. |
| [config.settings.kubectl.commands.resources](./values.yaml#L287) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps"]` | Defines which K8s resource are allowed. |
| [config.settings.kubectl.defaultNamespace](./values.yaml#L289) | string | `"default"` | Defines the default Namespace for executing BotKube `kubectl` commands. |
| [config.settings.kubectl.restrictAccess](./values.yaml#L291) | bool | `false` | If true, enables commands execution from configured channel only. |
| [config.settings.configwatcher](./values.yaml#L293) | bool | `true` | If true, restart the BotKube Pod on config changes. |
| [config.settings.upgradeNotifier](./values.yaml#L295) | bool | `true` | If true, notify about new BotKube releases. |
| [communications.existingSecretName](./values.yaml#L307) | string | `""` | Define existing Secret with communication settings. It MUST be in the `botkube` Namespace.  |
| [communications.slack.enabled](./values.yaml#L312) | bool | `false` | If yes, Slack bot is enabled. |
| [communications.slack.channel](./values.yaml#L314) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added BotKube and want to receive notifications in. |
| [communications.slack.token](./values.yaml#L316) | string | `"SLACK_API_TOKEN"` | Slack token. |
| [communications.slack.notiftype](./values.yaml#L318) | string | `"short"` | Define notification type that are sent. Possible values: `short`, `long`. |
| [communications.mattermost.enabled](./values.yaml#L323) | bool | `false` | If yes, Mattermost bot is enabled. |
| [communications.mattermost.botName](./values.yaml#L325) | string | `"BotKube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.mattermost.url](./values.yaml#L327) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.mattermost.token](./values.yaml#L329) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by BotKube user. |
| [communications.mattermost.team](./values.yaml#L331) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where BotKube is added. |
| [communications.mattermost.channel](./values.yaml#L334) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.mattermost.notiftype](./values.yaml#L336) | string | `"short"` | Define notification type that are sent. Possible values: `short`, `long`. |
| [communications.teams.enabled](./values.yaml#L341) | bool | `false` | If yes, MS Teams bot is enabled. |
| [communications.teams.appID](./values.yaml#L343) | string | `"APPLICATION_ID"` | The BotKube application ID generated while registering Bot to MS Teams. |
| [communications.teams.appPassword](./values.yaml#L345) | string | `"APPLICATION_PASSWORD"` | The BotKube application password generated while registering Bot to MS Teams. |
| [communications.teams.messagePath](./values.yaml#L347) | string | `"/bots/teams"` | The path in endpoint URL provided while registering BotKube to MS Teams. |
| [communications.teams.notiftype](./values.yaml#L349) | string | `"short"` | Define notification type that are sent. Possible values: `short`, `long`. |
| [communications.teams.port](./values.yaml#L351) | int | `3978` | The Service port for bot endpoint on BotKube container. |
| [communications.discord.enabled](./values.yaml#L356) | bool | `false` | If yes, Discord bot is enabled. |
| [communications.discord.token](./values.yaml#L358) | string | `"DISCORD_TOKEN"` | BotKube Bot Token. |
| [communications.discord.botid](./values.yaml#L360) | string | `"DISCORD_BOT_ID"` | BotKube Application Client ID. |
| [communications.discord.channel](./values.yaml#L363) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.discord.notiftype](./values.yaml#L365) | string | `"short"` | Define notification type that are sent. Possible values: `short`, `long`. |
| [communications.elasticsearch.enabled](./values.yaml#L370) | bool | `false` | If yes, Elasticsearch is enabled. |
| [communications.elasticsearch.awsSigning.enabled](./values.yaml#L374) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.elasticsearch.awsSigning.awsRegion](./values.yaml#L376) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.elasticsearch.awsSigning.roleArn](./values.yaml#L378) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.elasticsearch.server](./values.yaml#L380) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.elasticsearch.username](./values.yaml#L382) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.elasticsearch.password](./values.yaml#L384) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.elasticsearch.skipTLSVerify](./values.yaml#L387) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.elasticsearch.index](./values.yaml#L389) | object | `{"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}` | Elasticsearch index settings. |
| [communications.webhook.enabled](./values.yaml#L398) | bool | `false` | If yes, Elasticsearch is enabled. |
| [communications.webhook.url](./values.yaml#L400) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [service](./values.yaml#L403) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Service Settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L409) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Ingress settings to expose MS Teams endpoint. |
| [serviceMonitor](./values.yaml#L420) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L430) | object | `{}` | Extra annotations to pass to the BotKube Deployment. |
| [extraAnnotations](./values.yaml#L437) | object | `{}` | Extra annotations to pass to the BotKube Pod. |
| [priorityClassName](./values.yaml#L439) | string | `""` | Priority class name for the BotKube Pod. |
| [nameOverride](./values.yaml#L442) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L444) | string | `""` | fully override "botkube.fullname" template. |
| [resources](./values.yaml#L450) | object | `{}` | The BotKube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L462) | list | `[]` | Extra environment variables to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L474) | list | `[]` | Extra volumes to pass to the BotKube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L489) | list | `[]` | Extra volume mounts to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L507) | object | `{}` | Node labels for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L511) | list | `[]` | Tolerations for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L515) | object | `{}` | Affinity for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L519) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for BotKube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L528) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L531) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L533) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L536) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L563) | bool | `false` | If true, sending anonymous analytics is disabled. |
| [e2eTest.image.registry](./values.yaml#L569) | string | `"ghcr.io"` | Test runner image registry. |
| [e2eTest.image.repository](./values.yaml#L571) | string | `"kubeshop/botkube-test"` | Test runner image repository. |
| [e2eTest.image.pullPolicy](./values.yaml#L573) | string | `"IfNotPresent"` | Test runner image pull policy. |
| [e2eTest.image.tag](./values.yaml#L575) | string | `"v9.99.9-dev"` | Test runner image tag. Default tag is `appVersion` from Chart.yaml. |
| [e2eTest.deployment](./values.yaml#L577) | object | `{"waitTimeout":"3m"}` | Defines BotKube Deployment related data. |
| [e2eTest.slack.botName](./values.yaml#L582) | string | `"botkube"` | Name of the BotKube bot to interact with during the e2e tests. |
| [e2eTest.slack.testerAppToken](./values.yaml#L584) | string | `""` | Slack tester application token that interacts with BotKube bot. |
| [e2eTest.slack.additionalContextMessage](./values.yaml#L586) | string | `""` | Additional message that is sent by Tester. You can pass e.g. pull request number or source link where these tests are run from. |
| [e2eTest.slack.messageWaitTimeout](./values.yaml#L588) | string | `"1m"` | Message wait timeout. It defines how long we wait to ensure that notification were not sent when disabled. |
