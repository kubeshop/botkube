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

## Parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| [image.registry](./values.yaml#L14) | string | `"ghcr.io"` | BotKube container image registry. |
| [image.repository](./values.yaml#L16) | string | `"kubeshop/botkube"` | BotKube container image repository. |
| [image.pullPolicy](./values.yaml#L18) | string | `"IfNotPresent"` | BotKube container image pull policy. |
| [image.tag](./values.yaml#L20) | string | `"v9.99.9-dev"` | BotKube container image tag. Default tag is `appVersion` from Chart.yaml. |
| [podSecurityPolicy](./values.yaml#L24) | object | `{"enabled":false}` | Configures Pod Security Policy to allow BotKube to run in restricted clusters. [Ref doc](https://kubernetes.io/docs/concepts/policy/pod-security-policy/). |
| [securityContext](./values.yaml#L30) | object | Runs as a Non-Privileged user. | Configures security context to manage user Privileges in Pod. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). |
| [containerSecurityContext](./values.yaml#L36) | object | `{"allowPrivilegeEscalation":false,"privileged":false,"readOnlyRootFilesystem":true}` | Configures container security context. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container). |
| [kubeconfig.enabled](./values.yaml#L44) | bool | `false` | If true, enables overriding the Kubernetes auth. |
| [kubeconfig.base64Config](./values.yaml#L46) | string | `""` | A base64 encoded kubeconfig that will be stored in a Secret, mounted to the Pod, and specified in the KUBECONFIG environment variable. |
| [kubeconfig.existingSecret](./values.yaml#L51) | string | `""` | A Secret containing a kubeconfig to use.  |
| [log.level](./values.yaml#L56) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [log.disableColors](./values.yaml#L58) | bool | `false` | If true, disable ANSI colors in logging. |
| [config.resources](./values.yaml#L64) | list | Watch all built-in K8s kinds. | Describes the Kubernetes resources you want to watch. |
| [config.recommendations](./values.yaml#L268) | bool | `true` | If true, BotKube sends recommendations about the best practices for the created resource. |
| [config.ssl.enabled](./values.yaml#L273) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [config.ssl.existingSecretName](./values.yaml#L279) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [config.ssl.cert](./values.yaml#L282) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [config.settings.clustername](./values.yaml#L287) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [config.settings.kubectl.enabled](./values.yaml#L291) | bool | `false` | If true, enables `kubectl` commands execution. |
| [config.settings.kubectl.commands.verbs](./values.yaml#L295) | list | `["api-resources","api-versions","cluster-info","describe","diff","explain","get","logs","top","auth"]` | Configures which `kubectl` methods are allowed. |
| [config.settings.kubectl.commands.resources](./values.yaml#L297) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps"]` | Configures which K8s resource are allowed. |
| [config.settings.kubectl.defaultNamespace](./values.yaml#L299) | string | `"default"` | Configures the default Namespace for executing BotKube `kubectl` commands. |
| [config.settings.kubectl.restrictAccess](./values.yaml#L301) | bool | `false` | If true, enables commands execution from configured channel only. |
| [config.settings.configwatcher](./values.yaml#L303) | bool | `true` | If true, restarts the BotKube Pod on config changes. |
| [config.settings.upgradeNotifier](./values.yaml#L305) | bool | `true` | If true, notifies about new BotKube releases. |
| [communications.existingSecretName](./values.yaml#L317) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace.  |
| [communications.slack.enabled](./values.yaml#L322) | bool | `false` | If true, enables Slack bot. |
| [communications.slack.channel](./values.yaml#L324) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added BotKube and want to receive notifications in. |
| [communications.slack.token](./values.yaml#L326) | string | `"SLACK_API_TOKEN"` | Slack token. |
| [communications.slack.notiftype](./values.yaml#L328) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.mattermost.enabled](./values.yaml#L333) | bool | `false` | If true, enables Mattermost bot. |
| [communications.mattermost.botName](./values.yaml#L335) | string | `"BotKube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.mattermost.url](./values.yaml#L337) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.mattermost.token](./values.yaml#L339) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by BotKube user. |
| [communications.mattermost.team](./values.yaml#L341) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where BotKube is added. |
| [communications.mattermost.channel](./values.yaml#L344) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.mattermost.notiftype](./values.yaml#L346) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.teams.enabled](./values.yaml#L351) | bool | `false` | If true, enables MS Teams bot. |
| [communications.teams.appID](./values.yaml#L353) | string | `"APPLICATION_ID"` | The BotKube application ID generated while registering Bot to MS Teams. |
| [communications.teams.appPassword](./values.yaml#L355) | string | `"APPLICATION_PASSWORD"` | The BotKube application password generated while registering Bot to MS Teams. |
| [communications.teams.messagePath](./values.yaml#L357) | string | `"/bots/teams"` | The path in endpoint URL provided while registering BotKube to MS Teams. |
| [communications.teams.notiftype](./values.yaml#L359) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.teams.port](./values.yaml#L361) | int | `3978` | The Service port for bot endpoint on BotKube container. |
| [communications.discord.enabled](./values.yaml#L366) | bool | `false` | If true, enables Discord bot. |
| [communications.discord.token](./values.yaml#L368) | string | `"DISCORD_TOKEN"` | BotKube Bot Token. |
| [communications.discord.botid](./values.yaml#L370) | string | `"DISCORD_BOT_ID"` | BotKube Application Client ID. |
| [communications.discord.channel](./values.yaml#L373) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.discord.notiftype](./values.yaml#L375) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.elasticsearch.enabled](./values.yaml#L380) | bool | `false` | If true, enables Elasticsearch. |
| [communications.elasticsearch.awsSigning.enabled](./values.yaml#L384) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.elasticsearch.awsSigning.awsRegion](./values.yaml#L386) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.elasticsearch.awsSigning.roleArn](./values.yaml#L388) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.elasticsearch.server](./values.yaml#L390) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.elasticsearch.username](./values.yaml#L392) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.elasticsearch.password](./values.yaml#L394) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.elasticsearch.skipTLSVerify](./values.yaml#L397) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.elasticsearch.index](./values.yaml#L399) | object | `{"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}` | Configures Elasticsearch index settings. |
| [communications.webhook.enabled](./values.yaml#L408) | bool | `false` | If true, enables Webhook. |
| [communications.webhook.url](./values.yaml#L410) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [service](./values.yaml#L413) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L420) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L431) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L441) | object | `{}` | Extra annotations to pass to the BotKube Deployment. |
| [extraAnnotations](./values.yaml#L448) | object | `{}` | Extra annotations to pass to the BotKube Pod. |
| [priorityClassName](./values.yaml#L450) | string | `""` | Priority class name for the BotKube Pod. |
| [nameOverride](./values.yaml#L453) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L455) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L461) | object | `{}` | The BotKube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L473) | list | `[]` | Extra environment variables to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L485) | list | `[]` | Extra volumes to pass to the BotKube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L500) | list | `[]` | Extra volume mounts to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L518) | object | `{}` | Node labels for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L522) | list | `[]` | Tolerations for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L526) | object | `{}` | Affinity for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L530) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for BotKube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L539) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L542) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L544) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L547) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L575) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://botkube.io/privacy#privacy-policy). |
| [e2eTest.image.registry](./values.yaml#L581) | string | `"ghcr.io"` | Test runner image registry. |
| [e2eTest.image.repository](./values.yaml#L583) | string | `"kubeshop/botkube-test"` | Test runner image repository. |
| [e2eTest.image.pullPolicy](./values.yaml#L585) | string | `"IfNotPresent"` | Test runner image pull policy. |
| [e2eTest.image.tag](./values.yaml#L587) | string | `"v9.99.9-dev"` | Test runner image tag. Default tag is `appVersion` from Chart.yaml. |
| [e2eTest.deployment](./values.yaml#L589) | object | `{"waitTimeout":"3m"}` | Configures BotKube Deployment related data. |
| [e2eTest.slack.botName](./values.yaml#L594) | string | `"botkube"` | Name of the BotKube bot to interact with during the e2e tests. |
| [e2eTest.slack.testerAppToken](./values.yaml#L596) | string | `""` | Slack tester application token that interacts with BotKube bot. |
| [e2eTest.slack.additionalContextMessage](./values.yaml#L598) | string | `""` | Additional message that is sent by Tester. You can pass e.g. pull request number or source link where these tests are run from. |
| [e2eTest.slack.messageWaitTimeout](./values.yaml#L600) | string | `"1m"` | Message wait timeout. It defines how long we wait to ensure that notification were not sent when disabled. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine grained access. This is useful if you are looking to run BotKube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "<role_arn_to_assume>"
```
