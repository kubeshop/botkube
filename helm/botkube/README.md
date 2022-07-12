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

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working

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
| [kubeconfig.enabled](./values.yaml#L34) | bool | `false` | If true, enables overriding the kubernetes auth. |
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
| [config.settings.upgradeNotifier](./values.yaml#L295) | bool | `true` | If ture, notify about new BotKube releases. |
| [communications.existingSecretName](./values.yaml#L307) | string | `""` | Define existing Secret with communication settings. It MUST be in the `botkube` Namespace.  |
| [communications.slack](./values.yaml#L310) | object | `{"channel":"SLACK_CHANNEL","enabled":false,"notiftype":"short","token":"SLACK_API_TOKEN"}` | Settings for Slack. |
| [communications.mattermost](./values.yaml#L317) | object | `{"botName":"BotKube","channel":"MATTERMOST_CHANNEL","enabled":false,"notiftype":"short","team":"MATTERMOST_TEAM","token":"MATTERMOST_TOKEN","url":"MATTERMOST_SERVER_URL"}` | Settings for Mattermost. |
| [communications.teams](./values.yaml#L327) | object | `{"appID":"APPLICATION_ID","appPassword":"APPLICATION_PASSWORD","enabled":false,"messagePath":"/bots/teams","notiftype":"short","port":3978}` | Settings for MS Teams |
| [communications.discord](./values.yaml#L336) | object | `{"botid":"DISCORD_BOT_ID","channel":"DISCORD_CHANNEL_ID","enabled":false,"notiftype":"short","token":"DISCORD_TOKEN"}` | Settings for Discord. |
| [communications.elasticsearch](./values.yaml#L344) | object | `{"awsSigning":{"awsRegion":"us-east-1","enabled":false,"roleArn":""},"enabled":false,"index":{"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"},"password":"ELASTICSEARCH_PASSWORD","server":"ELASTICSEARCH_ADDRESS","skipTLSVerify":false,"username":"ELASTICSEARCH_USERNAME"}` | Settings for Elasticsearch. |
| [communications.webhook](./values.yaml#L362) | object | `{"enabled":false,"url":"WEBHOOK_URL"}` | Settings for Webhook. |
| [service](./values.yaml#L367) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Service Settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L373) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Ingress settings to expose MS Teams endpoint. |
| [serviceMonitor](./values.yaml#L384) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L396) | object | `{}` | Extra annotations to pass to the BotKube Deployment. |
| [replicaCount](./values.yaml#L399) | int | `1` | Number of BotKube pods to load balance between. |
| [extraAnnotations](./values.yaml#L401) | object | `{}` | Extra annotations to pass to the BotKube Pod. |
| [priorityClassName](./values.yaml#L403) | string | `""` | Priority class name for the BotKube Pod. |
| [nameOverride](./values.yaml#L406) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L408) | string | `""` | fully override "botkube.fullname" template. |
| [resources](./values.yaml#L414) | object | `{}` | We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L426) | list | `[]` | Extra environment variables to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L438) | list | `[]` | Extra volumes to pass to the BotKube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L453) | list | `[]` | Extra volume mounts to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L469) | object | `{}` |  |
| [tolerations](./values.yaml#L471) | list | `[]` |  |
| [affinity](./values.yaml#L473) | object | `{}` |  |
| [rbac.create](./values.yaml#L476) | bool | `true` |  |
| [rbac.rules[0].apiGroups[0]](./values.yaml#L478) | string | `"*"` |  |
| [rbac.rules[0].resources[0]](./values.yaml#L479) | string | `"*"` |  |
| [rbac.rules[0].verbs[0]](./values.yaml#L480) | string | `"get"` |  |
| [rbac.rules[0].verbs[1]](./values.yaml#L480) | string | `"watch"` |  |
| [rbac.rules[0].verbs[2]](./values.yaml#L480) | string | `"list"` |  |
| [serviceAccount.create](./values.yaml#L483) | bool | `true` |  |
| [serviceAccount.name](./values.yaml#L486) | string | `""` |  |
| [serviceAccount.annotations](./values.yaml#L488) | object | `{}` |  |
| [extraObjects](./values.yaml#L491) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L518) | bool | `false` |  |
| [e2eTest.image.registry](./values.yaml#L523) | string | `"ghcr.io"` |  |
| [e2eTest.image.repository](./values.yaml#L524) | string | `"kubeshop/botkube-test"` |  |
| [e2eTest.image.tag](./values.yaml#L525) | string | `"v9.99.9-dev"` |  |
| [e2eTest.image.pullPolicy](./values.yaml#L526) | string | `"IfNotPresent"` |  |
| [e2eTest.deployment.waitTimeout](./values.yaml#L528) | string | `"3m"` |  |
| [e2eTest.slack.botName](./values.yaml#L530) | string | `"botkube"` |  |
| [e2eTest.slack.testerAppToken](./values.yaml#L531) | string | `""` |  |
| [e2eTest.slack.additionalContextMessage](./values.yaml#L532) | string | `""` |  |
| [e2eTest.slack.messageWaitTimeout](./values.yaml#L533) | string | `"1m"` |  |
