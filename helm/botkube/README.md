# BotKube

![Version: v0.13.0](https://img.shields.io/badge/Version-v0.13.0-informational?style=flat-square) ![AppVersion: v0.13.0](https://img.shields.io/badge/AppVersion-v0.13.0-informational?style=flat-square)

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
| [sources](./values.yaml#L60) | object | See the `values.yaml` file for full object. | Map of sources. Source contains configuration for Kubernetes events and sending recommendations. The property name under `sources` object is an alias for a given configuration. You can define multiple sources configuration with different names. Key name is used as a binding reference.   |
| [sources.k8s-recommendation-events.kubernetes](./values.yaml#L64) | object | `{"recommendations":{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}}` | Describes Kubernetes source configuration. |
| [sources.k8s-recommendation-events.kubernetes.recommendations](./values.yaml#L66) | object | `{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}` | Describes configuration for various recommendation insights. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod](./values.yaml#L68) | object | `{"labelsSet":true,"noLatestImageTag":true}` | Recommendations for Pod Kubernetes resource. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod.noLatestImageTag](./values.yaml#L70) | bool | `true` | If true, notifies about Pod containers that use `latest` tag for images. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod.labelsSet](./values.yaml#L72) | bool | `true` | If true, notifies about Pod resources created without labels. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress](./values.yaml#L74) | object | `{"backendServiceValid":true,"tlsSecretValid":true}` | Recommendations for Ingress Kubernetes resource. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress.backendServiceValid](./values.yaml#L76) | bool | `true` | If true, notifies about Ingress resources with invalid backend service reference. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress.tlsSecretValid](./values.yaml#L78) | bool | `true` | If true, notifies about Ingress resources with invalid TLS secret reference. |
| [sources.k8s-all-events.kubernetes](./values.yaml#L83) | object | `{"events":["create","delete","error"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"v1/services"},{"name":"networking.k8s.io/v1/ingresses"},{"name":"v1/nodes"},{"name":"v1/namespaces"},{"name":"v1/persistentvolumes"},{"name":"v1/persistentvolumeclaims"},{"name":"v1/configmaps"},{"name":"rbac.authorization.k8s.io/v1/roles"},{"name":"rbac.authorization.k8s.io/v1/rolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterroles"},{"events":["create","update","delete","error"],"name":"apps/v1/daemonsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.numberReady"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"batch/v1/jobs","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.conditions[*].type"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"apps/v1/deployments","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.availableReplicas"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"apps/v1/statefulsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.readyReplicas"],"includeDiff":true}}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.kubernetes.namespaces](./values.yaml#L87) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-all-events.kubernetes.events](./values.yaml#L101) | list | `["create","delete","error"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-all-events.kubernetes.resources](./values.yaml#L108) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-events.kubernetes](./values.yaml#L192) | object | `{"events":["error"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"v1/services"},{"name":"networking.k8s.io/v1/ingresses"},{"name":"v1/nodes"},{"name":"v1/namespaces"},{"name":"v1/persistentvolumes"},{"name":"v1/persistentvolumeclaims"},{"name":"v1/configmaps"},{"name":"rbac.authorization.k8s.io/v1/roles"},{"name":"rbac.authorization.k8s.io/v1/rolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterroles"},{"name":"apps/v1/deployments"},{"name":"apps/v1/statefulsets"},{"name":"apps/v1/daemonsets"},{"name":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-err-events.kubernetes.namespaces](./values.yaml#L196) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.kubernetes.events](./values.yaml#L201) | list | `["error"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-err-events.kubernetes.resources](./values.yaml#L206) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [filters](./values.yaml#L228) | object | See the `values.yaml` file for full object. | Filter settings for various sources. Currently, all filters are globally enabled or disabled. You can enable or disable filters with `@BotKube filters` commands. |
| [filters.kubernetes.objectAnnotationChecker](./values.yaml#L231) | bool | `true` | If true, enables support for `botkube.io/disable` and `botkube.io/channel` resource annotations. |
| [filters.kubernetes.nodeEventsChecker](./values.yaml#L233) | bool | `true` | If true, filters out Node-related events that are not important. |
| [executors](./values.yaml#L241) | object | See the `values.yaml` file for full object. | Map of executors. Executor contains configuration for running `kubectl` commands. The property name under `executors` is an alias for a given configuration. You can define multiple executor configurations with different names. Key name is used as a binding reference.   |
| [executors.kubectl-read-only.kubectl.namespaces.include](./values.yaml#L249) | list | `[".*"]` | List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions:  `- ".*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.namespaces.exclude](./values.yaml#L254) | list | `[]` | List of ignored Kubernetes Namespace. It can also contain a regex expressions:  `- "test-.*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.enabled](./values.yaml#L256) | bool | `false` | If true, enables `kubectl` commands execution. |
| [executors.kubectl-read-only.kubectl.commands.verbs](./values.yaml#L260) | list | `["api-resources","api-versions","cluster-info","describe","diff","explain","get","logs","top","auth"]` | Configures which `kubectl` methods are allowed. |
| [executors.kubectl-read-only.kubectl.commands.resources](./values.yaml#L262) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps"]` | Configures which K8s resource are allowed. |
| [executors.kubectl-read-only.kubectl.defaultNamespace](./values.yaml#L264) | string | `"default"` | Configures the default Namespace for executing BotKube `kubectl` commands. If not set, uses the 'default'. |
| [executors.kubectl-read-only.kubectl.restrictAccess](./values.yaml#L266) | bool | `false` | If true, enables commands execution from configured channel only. |
| [existingCommunicationsSecretName](./values.yaml#L277) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace. To reload BotKube once it changes, add label `botkube.io/config-watch: "true"`.  |
| [communications](./values.yaml#L284) | object | See the `values.yaml` file for full object. | Map of communication groups. Communication group contains settings for multiple communication platforms. The property name under `communications` object is an alias for a given configuration group. You can define multiple communication groups with different names.   |
| [communications.default-group.slack.enabled](./values.yaml#L289) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.slack.channels](./values.yaml#L293) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.slack.channels.default.name](./values.yaml#L296) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added BotKube and want to receive notifications in. |
| [communications.default-group.slack.channels.default.notification.disabled](./values.yaml#L299) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@BotKube` command anytime. |
| [communications.default-group.slack.channels.default.bindings.executors](./values.yaml#L302) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.slack.channels.default.bindings.sources](./values.yaml#L305) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.slack.token](./values.yaml#L309) | string | `""` | Slack token. |
| [communications.default-group.slack.notification.type](./values.yaml#L312) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.socketSlack.enabled](./values.yaml#L317) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.socketSlack.channels](./values.yaml#L321) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.socketSlack.channels.default.name](./values.yaml#L324) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added BotKube and want to receive notifications in. |
| [communications.default-group.socketSlack.channels.default.bindings.executors](./values.yaml#L327) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.socketSlack.channels.default.bindings.sources](./values.yaml#L330) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.socketSlack.botToken](./values.yaml#L335) | string | `""` | Slack bot token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.appToken](./values.yaml#L338) | string | `""` | Slack app-level token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.notification.type](./values.yaml#L341) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.mattermost.enabled](./values.yaml#L345) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L347) | string | `"BotKube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L349) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L351) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by BotKube user. |
| [communications.default-group.mattermost.team](./values.yaml#L353) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where BotKube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L357) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"MATTERMOST_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L361) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.default-group.mattermost.channels.default.notification.disabled](./values.yaml#L364) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@BotKube` command anytime. |
| [communications.default-group.mattermost.channels.default.bindings.executors](./values.yaml#L367) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.mattermost.channels.default.bindings.sources](./values.yaml#L370) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.mattermost.notification.type](./values.yaml#L375) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.teams.enabled](./values.yaml#L380) | bool | `false` | If true, enables MS Teams bot. |
| [communications.default-group.teams.botName](./values.yaml#L382) | string | `"BotKube"` | The Bot name set while registering Bot to MS Teams. |
| [communications.default-group.teams.appID](./values.yaml#L384) | string | `"APPLICATION_ID"` | The BotKube application ID generated while registering Bot to MS Teams. |
| [communications.default-group.teams.appPassword](./values.yaml#L386) | string | `"APPLICATION_PASSWORD"` | The BotKube application password generated while registering Bot to MS Teams. |
| [communications.default-group.teams.bindings.executors](./values.yaml#L389) | list | `["kubectl-read-only"]` | Executor bindings apply to all MS Teams channels where BotKube has access to. |
| [communications.default-group.teams.bindings.sources](./values.yaml#L392) | list | `["k8s-err-events","k8s-recommendation-events"]` | Source bindings apply to all channels which have notification turned on with `@BotKube notifier start` command. |
| [communications.default-group.teams.messagePath](./values.yaml#L396) | string | `"/bots/teams"` | The path in endpoint URL provided while registering BotKube to MS Teams. |
| [communications.default-group.teams.port](./values.yaml#L398) | int | `3978` | The Service port for bot endpoint on BotKube container. |
| [communications.default-group.discord.enabled](./values.yaml#L403) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L405) | string | `"DISCORD_TOKEN"` | BotKube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L407) | string | `"DISCORD_BOT_ID"` | BotKube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L411) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"id":"DISCORD_CHANNEL_ID","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L415) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving BotKube alerts. The BotKube user needs to be added to it. |
| [communications.default-group.discord.channels.default.notification.disabled](./values.yaml#L418) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@BotKube` command anytime. |
| [communications.default-group.discord.channels.default.bindings.executors](./values.yaml#L421) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.discord.channels.default.bindings.sources](./values.yaml#L424) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.discord.notification.type](./values.yaml#L429) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L434) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L438) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L440) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L442) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L444) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L446) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L448) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L451) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.indices](./values.yaml#L455) | object | `{"default":{"bindings":{"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L458) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.elasticsearch.indices.default.bindings.sources](./values.yaml#L464) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given index. |
| [communications.default-group.webhook.enabled](./values.yaml#L471) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L473) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [communications.default-group.webhook.bindings.sources](./values.yaml#L476) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for the webhook. |
| [settings.clusterName](./values.yaml#L483) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.lifecycleServer](./values.yaml#L486) | object | `{"enabled":true,"port":2113}` | Server configuration which exposes functionality related to the app lifecycle. |
| [settings.upgradeNotifier](./values.yaml#L490) | bool | `true` | If true, notifies about new BotKube releases. |
| [settings.log.level](./values.yaml#L494) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L496) | bool | `false` | If true, disable ANSI colors in logging. |
| [settings.systemConfigMap](./values.yaml#L499) | object | `{"name":"botkube-system"}` | BotKube's system ConfigMap where internal data is stored. |
| [settings.persistentConfig](./values.yaml#L504) | object | `{"runtime":{"configMap":{"annotations":{},"name":"botkube-runtime-config"},"fileName":"_runtime_state.yaml"},"startup":{"configMap":{"annotations":{},"name":"botkube-startup-config"},"fileName":"_startup_state.yaml"}}` | Persistent config contains ConfigMap where persisted configuration is stored. The persistent configuration is evaluated from both chart upgrade and BotKube commands used in runtime. |
| [ssl.enabled](./values.yaml#L519) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L525) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L528) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L531) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L538) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L549) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L559) | object | `{}` | Extra annotations to pass to the BotKube Deployment. |
| [extraAnnotations](./values.yaml#L566) | object | `{}` | Extra annotations to pass to the BotKube Pod. |
| [extraLabels](./values.yaml#L568) | object | `{}` | Extra labels to pass to the BotKube Pod. |
| [priorityClassName](./values.yaml#L570) | string | `""` | Priority class name for the BotKube Pod. |
| [nameOverride](./values.yaml#L573) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L575) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L581) | object | `{}` | The BotKube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L593) | list | `[]` | Extra environment variables to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L605) | list | `[]` | Extra volumes to pass to the BotKube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L620) | list | `[]` | Extra volume mounts to pass to the BotKube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L638) | object | `{}` | Node labels for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L642) | list | `[]` | Tolerations for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L646) | object | `{}` | Affinity for BotKube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L650) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for BotKube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L659) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L662) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L664) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L667) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L695) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://botkube.io/privacy#privacy-policy). |
| [configWatcher.enabled](./values.yaml#L700) | bool | `true` | If true, restarts the BotKube Pod on config changes. |
| [configWatcher.tmpDir](./values.yaml#L702) | string | `"/tmp/watched-cfg/"` | Directory, where watched configuration resources are stored. |
| [configWatcher.initialSyncTimeout](./values.yaml#L704) | string | `"10s"` | Timeout for the initial Config Watcher sync. If set to 0, waiting for Config Watcher sync will be skipped. |
| [configWatcher.image.registry](./values.yaml#L707) | string | `"ghcr.io"` | Config watcher image registry. |
| [configWatcher.image.repository](./values.yaml#L709) | string | `"kubeshop/k8s-sidecar"` | Config watcher image repository. |
| [configWatcher.image.tag](./values.yaml#L711) | string | `"ignore-initial-events"` | Config watcher image tag. |
| [configWatcher.image.pullPolicy](./values.yaml#L713) | string | `"IfNotPresent"` | Config watcher image pull policy. |
| [e2eTest.image.registry](./values.yaml#L719) | string | `"ghcr.io"` | Test runner image registry. |
| [e2eTest.image.repository](./values.yaml#L721) | string | `"kubeshop/botkube-test"` | Test runner image repository. |
| [e2eTest.image.pullPolicy](./values.yaml#L723) | string | `"IfNotPresent"` | Test runner image pull policy. |
| [e2eTest.image.tag](./values.yaml#L725) | string | `"v9.99.9-dev"` | Test runner image tag. Default tag is `appVersion` from Chart.yaml. |
| [e2eTest.deployment](./values.yaml#L727) | object | `{"waitTimeout":"3m"}` | Configures BotKube Deployment related data. |
| [e2eTest.slack.botName](./values.yaml#L732) | string | `"botkube"` | Name of the BotKube bot to interact with during the e2e tests. |
| [e2eTest.slack.testerName](./values.yaml#L734) | string | `"botkube_tester"` | Name of the BotKube Tester bot that sends messages during the e2e tests. |
| [e2eTest.slack.testerAppToken](./values.yaml#L736) | string | `""` | Slack tester application token that interacts with BotKube bot. |
| [e2eTest.slack.additionalContextMessage](./values.yaml#L738) | string | `""` | Additional message that is sent by Tester. You can pass e.g. pull request number or source link where these tests are run from. |
| [e2eTest.slack.messageWaitTimeout](./values.yaml#L740) | string | `"1m"` | Message wait timeout. It defines how long we wait to ensure that notification were not sent when disabled. |
| [e2eTest.discord.botName](./values.yaml#L743) | string | `"botkube"` | Name of the BotKube bot to interact with during the e2e tests. |
| [e2eTest.discord.testerName](./values.yaml#L745) | string | `"botkube_tester"` | Name of the BotKube Tester bot that sends messages during the e2e tests. |
| [e2eTest.discord.guildID](./values.yaml#L747) | string | `""` | Discord Guild ID (discord server ID) used to run e2e tests |
| [e2eTest.discord.testerAppToken](./values.yaml#L749) | string | `""` | Discord tester application token that interacts with BotKube bot. |
| [e2eTest.discord.additionalContextMessage](./values.yaml#L751) | string | `""` | Additional message that is sent by Tester. You can pass e.g. pull request number or source link where these tests are run from. |
| [e2eTest.discord.messageWaitTimeout](./values.yaml#L753) | string | `"1m"` | Message wait timeout. It defines how long we wait to ensure that notification were not sent when disabled. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine grained access. This is useful if you are looking to run BotKube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "<role_arn_to_assume>"
```
