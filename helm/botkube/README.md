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
| [sources](./values.yaml#L58) | object | `{"k8s-events":{"kubernetes":{"resources":[{"events":["create","delete","error"],"name":"v1/pods","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/services","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","update","delete","error"],"name":"apps/v1/deployments","namespaces":{"ignore":[null],"include":["all"]},"updateSetting":{"fields":["spec.template.spec.containers[*].image","status.availableReplicas"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"apps/v1/statefulsets","namespaces":{"ignore":[null],"include":["all"]},"updateSetting":{"fields":["spec.template.spec.containers[*].image","status.readyReplicas"],"includeDiff":true}},{"events":["create","delete","error"],"name":"networking.k8s.io/v1/ingresses","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/nodes","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/namespaces","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/persistentvolumes","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/persistentvolumeclaims","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"v1/configmaps","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","update","delete","error"],"name":"apps/v1/daemonsets","namespaces":{"ignore":[null],"include":["all"]},"updateSetting":{"fields":["spec.template.spec.containers[*].image","status.numberReady"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"batch/v1/jobs","namespaces":{"ignore":[null],"include":["all"]},"updateSetting":{"fields":["spec.template.spec.containers[*].image","status.conditions[*].type"],"includeDiff":true}},{"events":["create","delete","error"],"name":"rbac.authorization.k8s.io/v1/roles","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"rbac.authorization.k8s.io/v1/rolebindings","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"rbac.authorization.k8s.io/v1/clusterrolebindings","namespaces":{"ignore":[null],"include":["all"]}},{"events":["create","delete","error"],"name":"rbac.authorization.k8s.io/v1/clusterroles","namespaces":{"ignore":[null],"include":["all"]}}]},"recommendations":true}}` | Map of enabled sources. The `sources` property name is an alias for a given configuration. Key name used as a binding reference.   |
| [sources.k8s-events.recommendations](./values.yaml#L62) | bool | `true` | If true, BotKube sends recommendations about the best practices for the created resource. |
| [sources.k8s-events.kubernetes.resources](./values.yaml#L67) | list | Watch all built-in K8s kinds. | Describes the Kubernetes resources you want to watch. |
| [executors.kubectl-read-only.kubectl.namespaces.include](./values.yaml#L280) | list | `["all"]` | List of allowed Kubernetes Namespaces for command execution. The "all" keyword, allows all the Namespaces. |
| [executors.kubectl-read-only.kubectl.namespaces.ignore](./values.yaml#L285) | list | `[]` | List of ignored Kubernetes Namespace. It can be used only when `include: ["all"]`. It can contain a wildcard (*) that expands to zero or more arbitrary characters Example: `ignore: ["x", "y", "secret-ns-*"]` |
| [executors.kubectl-read-only.kubectl.enabled](./values.yaml#L287) | bool | `false` | If true, enables `kubectl` commands execution. |
| [executors.kubectl-read-only.kubectl.commands.verbs](./values.yaml#L291) | list | `["api-resources","api-versions","cluster-info","describe","diff","explain","get","logs","top","auth"]` | Configures which `kubectl` methods are allowed. |
| [executors.kubectl-read-only.kubectl.commands.resources](./values.yaml#L293) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps"]` | Configures which K8s resource are allowed. |
| [executors.kubectl-read-only.kubectl.defaultNamespace](./values.yaml#L295) | string | `"default"` | Configures the default Namespace for executing BotKube `kubectl` commands. |
| [executors.kubectl-read-only.kubectl.restrictAccess](./values.yaml#L297) | bool | `false` | If true, enables commands execution from configured channel only. |
| [existingCommunicationsSecretName](./values.yaml#L307) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace.  |
| [communications.default-group.slack.enabled](./values.yaml#L317) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.slack.channels](./values.yaml#L321) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The `channels` property name is an alias for a given configuration.   |
| [communications.default-group.slack.channels.default.name](./values.yaml#L324) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added BotKube and want to receive notifications in. |
| [communications.default-group.slack.token](./values.yaml#L331) | string | `"SLACK_API_TOKEN"` | Slack token. |
| [communications.default-group.slack.notification.type](./values.yaml#L334) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.mattermost.enabled](./values.yaml#L339) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L341) | string | `"BotKube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L343) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L345) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by BotKube user. |
| [communications.default-group.mattermost.team](./values.yaml#L347) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where BotKube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L351) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-events"]},"name":"MATTERMOST_CHANNEL"}}` | Map of configured channels. The `channels` property name is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L355) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.default-group.mattermost.notification.type](./values.yaml#L363) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.teams.enabled](./values.yaml#L368) | bool | `false` | If true, enables MS Teams bot. |
| [communications.default-group.teams.botName](./values.yaml#L370) | string | `"BotKube"` | The Bot name set while registering Bot to MS Teams. |
| [communications.default-group.teams.appID](./values.yaml#L372) | string | `"APPLICATION_ID"` | The BotKube application ID generated while registering Bot to MS Teams. |
| [communications.default-group.teams.appPassword](./values.yaml#L374) | string | `"APPLICATION_PASSWORD"` | The BotKube application password generated while registering Bot to MS Teams. |
| [communications.default-group.teams.messagePath](./values.yaml#L376) | string | `"/bots/teams"` | The path in endpoint URL provided while registering BotKube to MS Teams. |
| [communications.default-group.teams.notification.type](./values.yaml#L379) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.teams.port](./values.yaml#L381) | int | `3978` | The Service port for bot endpoint on BotKube container. |
| [communications.default-group.discord.enabled](./values.yaml#L386) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L388) | string | `"DISCORD_TOKEN"` | BotKube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L390) | string | `"DISCORD_BOT_ID"` | BotKube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L394) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-events"]},"id":"DISCORD_CHANNEL_ID"}}` | Map of configured channels. The `channels` property name is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L398) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.default-group.discord.notification.type](./values.yaml#L406) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L411) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L415) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L417) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L419) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L421) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L423) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L425) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L428) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.indices](./values.yaml#L432) | object | `{"default":{"bindings":{"sources":["k8s-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L435) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.webhook.enabled](./values.yaml#L446) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L448) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [settings.clusterName](./values.yaml#L453) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.configWatcher](./values.yaml#L455) | bool | `true` | If true, restarts the BotKube Pod on config changes. |
| [settings.upgradeNotifier](./values.yaml#L457) | bool | `true` | If true, notifies about new BotKube releases. |
| [settings.log.level](./values.yaml#L461) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L463) | bool | `false` | If true, disable ANSI colors in logging. |
| [ssl.enabled](./values.yaml#L468) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L474) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L477) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L480) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L487) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L498) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L508) | object | `{}` | Extra annotations to pass to the BotKube Deployment. |
| [extraAnnotations](./values.yaml#L515) | object | `{}` | Extra annotations to pass to the BotKube Pod. |
| [extraLabels](./values.yaml#L517) | object | `{}` | Extra labels to pass to the BotKube Pod. |
| [priorityClassName](./values.yaml#L519) | string | `""` | Priority class name for the BotKube Pod. |
| [nameOverride](./values.yaml#L522) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L524) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L530) | object | `{}` | The BotKube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L542) | list | `[]` | Extra environment variables to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L554) | list | `[]` | Extra volumes to pass to the BotKube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L569) | list | `[]` | Extra volume mounts to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L587) | object | `{}` | Node labels for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L591) | list | `[]` | Tolerations for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L595) | object | `{}` | Affinity for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L599) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for BotKube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L608) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L611) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L613) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L616) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L644) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://botkube.io/privacy#privacy-policy). |
| [e2eTest.image.registry](./values.yaml#L650) | string | `"ghcr.io"` | Test runner image registry. |
| [e2eTest.image.repository](./values.yaml#L652) | string | `"kubeshop/botkube-test"` | Test runner image repository. |
| [e2eTest.image.pullPolicy](./values.yaml#L654) | string | `"IfNotPresent"` | Test runner image pull policy. |
| [e2eTest.image.tag](./values.yaml#L656) | string | `"v9.99.9-dev"` | Test runner image tag. Default tag is `appVersion` from Chart.yaml. |
| [e2eTest.deployment](./values.yaml#L658) | object | `{"waitTimeout":"3m"}` | Configures BotKube Deployment related data. |
| [e2eTest.slack.botName](./values.yaml#L663) | string | `"botkube"` | Name of the BotKube bot to interact with during the e2e tests. |
| [e2eTest.slack.testerAppToken](./values.yaml#L665) | string | `""` | Slack tester application token that interacts with BotKube bot. |
| [e2eTest.slack.additionalContextMessage](./values.yaml#L667) | string | `""` | Additional message that is sent by Tester. You can pass e.g. pull request number or source link where these tests are run from. |
| [e2eTest.slack.messageWaitTimeout](./values.yaml#L669) | string | `"1m"` | Message wait timeout. It defines how long we wait to ensure that notification were not sent when disabled. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine grained access. This is useful if you are looking to run BotKube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "<role_arn_to_assume>"
```
