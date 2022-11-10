# Botkube

![Version: v9.99.9-dev](https://img.shields.io/badge/Version-v9.99.9--dev-informational?style=flat-square) ![AppVersion: v9.99.9-dev](https://img.shields.io/badge/AppVersion-v9.99.9--dev-informational?style=flat-square)

Controller for the Botkube Slack app which helps you monitor your Kubernetes cluster, debug deployments and run specific checks on resources in the cluster.

**Homepage:** <https://botkube.io>

## Maintainers

| Name | Email  |
| ---- | ------ |
| Botkube Dev Team | <dev-team@botkube.io> |

## Source Code

* <https://github.com/kubeshop/botkube>

## Parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| [image.registry](./values.yaml#L14) | string | `"ghcr.io"` | Botkube container image registry. |
| [image.repository](./values.yaml#L16) | string | `"kubeshop/botkube"` | Botkube container image repository. |
| [image.pullPolicy](./values.yaml#L18) | string | `"IfNotPresent"` | Botkube container image pull policy. |
| [image.tag](./values.yaml#L20) | string | `"v9.99.9-dev"` | Botkube container image tag. Default tag is `appVersion` from Chart.yaml. |
| [podSecurityPolicy](./values.yaml#L24) | object | `{"enabled":false}` | Configures Pod Security Policy to allow Botkube to run in restricted clusters. [Ref doc](https://kubernetes.io/docs/concepts/policy/pod-security-policy/). |
| [securityContext](./values.yaml#L30) | object | Runs as a Non-Privileged user. | Configures security context to manage user Privileges in Pod. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). |
| [containerSecurityContext](./values.yaml#L36) | object | `{"allowPrivilegeEscalation":false,"privileged":false,"readOnlyRootFilesystem":true}` | Configures container security context. [Ref doc](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container). |
| [kubeconfig.enabled](./values.yaml#L44) | bool | `false` | If true, enables overriding the Kubernetes auth. |
| [kubeconfig.base64Config](./values.yaml#L46) | string | `""` | A base64 encoded kubeconfig that will be stored in a Secret, mounted to the Pod, and specified in the KUBECONFIG environment variable. |
| [kubeconfig.existingSecret](./values.yaml#L51) | string | `""` | A Secret containing a kubeconfig to use.  |
| [actions](./values.yaml#L58) | object | See the `values.yaml` file for full object. | Map of actions. Action contains configuration for automations based on observed events. The property name under `actions` object is an alias for a given configuration. You can define multiple actions configuration with different names.   |
| [actions.show-created-resource.enabled](./values.yaml#L61) | bool | `false` | If true, enables the action. |
| [actions.show-created-resource.displayName](./values.yaml#L63) | string | `"Display created resource"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.show-created-resource.command](./values.yaml#L66) | string | `"kubectl describe {{ .Event.TypeMeta.Kind | lower }}{{ if .Event.Namespace -}} -n {{ .Event.Namespace }}{{- end }} {{ .Event.Name }}"` | A text value denoting the command run by this action, may contain even based templated values. The executor is inferred directly from the command, e.g. here we require a kubectl executor |
| [actions.show-created-resource.bindings](./values.yaml#L69) | object | `{"executors":["kubectl-read-only"],"sources":["k8s-create-events"]}` | Bindings for a given action. |
| [actions.show-created-resource.bindings.sources](./values.yaml#L71) | list | `["k8s-create-events"]` | Sources of events that trigger a given action. |
| [actions.show-created-resource.bindings.executors](./values.yaml#L74) | list | `["kubectl-read-only"]` | Executors configuration for a given automation. |
| [actions.show-logs-on-error.enabled](./values.yaml#L78) | bool | `false` | If true, enables the action. |
| [actions.show-logs-on-error.displayName](./values.yaml#L81) | string | `"Show logs on error"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.show-logs-on-error.command](./values.yaml#L84) | string | `"kubectl logs {{ .Event.TypeMeta.Kind | lower }}/{{ .Event.Name }} -n {{ .Event.Namespace }}"` | A text value denoting the command run by this action, may contain even based templated values. The executor is inferred directly from the command, e.g. here we require a kubectl executor |
| [actions.show-logs-on-error.bindings](./values.yaml#L87) | object | `{"executors":["kubectl-read-only"],"sources":["k8s-err-with-logs-events"]}` | Bindings for a given action. |
| [actions.show-logs-on-error.bindings.sources](./values.yaml#L89) | list | `["k8s-err-with-logs-events"]` | Sources of events that trigger a given action. |
| [actions.show-logs-on-error.bindings.executors](./values.yaml#L92) | list | `["kubectl-read-only"]` | Executors configuration for a given automation. |
| [sources](./values.yaml#L101) | object | See the `values.yaml` file for full object. | Map of sources. Source contains configuration for Kubernetes events and sending recommendations. The property name under `sources` object is an alias for a given configuration. You can define multiple sources configuration with different names. Key name is used as a binding reference.   |
| [sources.k8s-recommendation-events.kubernetes](./values.yaml#L105) | object | `{"recommendations":{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}}` | Describes Kubernetes source configuration. |
| [sources.k8s-recommendation-events.kubernetes.recommendations](./values.yaml#L107) | object | `{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}` | Describes configuration for various recommendation insights. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod](./values.yaml#L109) | object | `{"labelsSet":true,"noLatestImageTag":true}` | Recommendations for Pod Kubernetes resource. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod.noLatestImageTag](./values.yaml#L111) | bool | `true` | If true, notifies about Pod containers that use `latest` tag for images. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.pod.labelsSet](./values.yaml#L113) | bool | `true` | If true, notifies about Pod resources created without labels. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress](./values.yaml#L115) | object | `{"backendServiceValid":true,"tlsSecretValid":true}` | Recommendations for Ingress Kubernetes resource. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress.backendServiceValid](./values.yaml#L117) | bool | `true` | If true, notifies about Ingress resources with invalid backend service reference. |
| [sources.k8s-recommendation-events.kubernetes.recommendations.ingress.tlsSecretValid](./values.yaml#L119) | bool | `true` | If true, notifies about Ingress resources with invalid TLS secret reference. |
| [sources.k8s-all-events.kubernetes](./values.yaml#L124) | object | `{"events":["create","delete","error"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"v1/services"},{"name":"networking.k8s.io/v1/ingresses"},{"name":"v1/nodes"},{"name":"v1/namespaces"},{"name":"v1/persistentvolumes"},{"name":"v1/persistentvolumeclaims"},{"name":"v1/configmaps"},{"name":"rbac.authorization.k8s.io/v1/roles"},{"name":"rbac.authorization.k8s.io/v1/rolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterroles"},{"events":["create","update","delete","error"],"name":"apps/v1/daemonsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.numberReady"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"batch/v1/jobs","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.conditions[*].type"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"apps/v1/deployments","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.availableReplicas"],"includeDiff":true}},{"events":["create","update","delete","error"],"name":"apps/v1/statefulsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.readyReplicas"],"includeDiff":true}}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.kubernetes.namespaces](./values.yaml#L128) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-all-events.kubernetes.events](./values.yaml#L142) | list | `["create","delete","error"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-all-events.kubernetes.resources](./values.yaml#L149) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-events.kubernetes](./values.yaml#L233) | object | `{"events":["error"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"v1/services"},{"name":"networking.k8s.io/v1/ingresses"},{"name":"v1/nodes"},{"name":"v1/namespaces"},{"name":"v1/persistentvolumes"},{"name":"v1/persistentvolumeclaims"},{"name":"v1/configmaps"},{"name":"rbac.authorization.k8s.io/v1/roles"},{"name":"rbac.authorization.k8s.io/v1/rolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"name":"rbac.authorization.k8s.io/v1/clusterroles"},{"name":"apps/v1/deployments"},{"name":"apps/v1/statefulsets"},{"name":"apps/v1/daemonsets"},{"name":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-err-events.kubernetes.namespaces](./values.yaml#L237) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.kubernetes.events](./values.yaml#L242) | list | `["error"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-err-events.kubernetes.resources](./values.yaml#L247) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-with-logs-events.kubernetes](./values.yaml#L268) | object | `{"events":["error"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"apps/v1/deployments"},{"name":"apps/v1/statefulsets"},{"name":"apps/v1/daemonsets"},{"name":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-err-with-logs-events.kubernetes.namespaces](./values.yaml#L272) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-with-logs-events.kubernetes.events](./values.yaml#L277) | list | `["error"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-err-with-logs-events.kubernetes.resources](./values.yaml#L282) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-create-events.kubernetes](./values.yaml#L294) | object | `{"events":["create"],"namespaces":{"include":[".*"]},"resources":[{"name":"v1/pods"},{"name":"v1/services"},{"name":"networking.k8s.io/v1/ingresses"},{"name":"v1/nodes"},{"name":"v1/namespaces"},{"name":"v1/configmaps"},{"name":"apps/v1/deployments"},{"name":"apps/v1/statefulsets"},{"name":"apps/v1/daemonsets"},{"name":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-create-events.kubernetes.namespaces](./values.yaml#L298) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-create-events.kubernetes.events](./values.yaml#L303) | list | `["create"]` | Describes events for every Kubernetes resources you want to watch or exclude. These events are applied to every resource specified in the resources list. However, every specified resource can override this by using its own events object. |
| [sources.k8s-create-events.kubernetes.resources](./values.yaml#L308) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [filters](./values.yaml#L324) | object | See the `values.yaml` file for full object. | Filter settings for various sources. Currently, all filters are globally enabled or disabled. You can enable or disable filters with `@Botkube filters` commands. |
| [filters.kubernetes.objectAnnotationChecker](./values.yaml#L327) | bool | `true` | If true, enables support for `botkube.io/disable` and `botkube.io/channel` resource annotations. |
| [filters.kubernetes.nodeEventsChecker](./values.yaml#L329) | bool | `true` | If true, filters out Node-related events that are not important. |
| [executors](./values.yaml#L337) | object | See the `values.yaml` file for full object. | Map of executors. Executor contains configuration for running `kubectl` commands. The property name under `executors` is an alias for a given configuration. You can define multiple executor configurations with different names. Key name is used as a binding reference.   |
| [executors.kubectl-read-only.kubectl.namespaces.include](./values.yaml#L345) | list | `[".*"]` | List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions:  `- ".*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.namespaces.exclude](./values.yaml#L350) | list | `[]` | List of ignored Kubernetes Namespace. It can also contain a regex expressions:  `- "test-.*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.enabled](./values.yaml#L352) | bool | `false` | If true, enables `kubectl` commands execution. |
| [executors.kubectl-read-only.kubectl.commands.verbs](./values.yaml#L356) | list | `["api-resources","api-versions","cluster-info","describe","explain","get","logs","top"]` | Configures which `kubectl` methods are allowed. |
| [executors.kubectl-read-only.kubectl.commands.resources](./values.yaml#L358) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps","services","ingresses"]` | Configures which K8s resource are allowed. |
| [executors.kubectl-read-only.kubectl.defaultNamespace](./values.yaml#L360) | string | `"default"` | Configures the default Namespace for executing Botkube `kubectl` commands. If not set, uses the 'default'. |
| [executors.kubectl-read-only.kubectl.restrictAccess](./values.yaml#L362) | bool | `false` | If true, enables commands execution from configured channel only. |
| [existingCommunicationsSecretName](./values.yaml#L373) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace. To reload Botkube once it changes, add label `botkube.io/config-watch: "true"`.  |
| [communications](./values.yaml#L380) | object | See the `values.yaml` file for full object. | Map of communication groups. Communication group contains settings for multiple communication platforms. The property name under `communications` object is an alias for a given configuration group. You can define multiple communication groups with different names.   |
| [communications.default-group.slack.enabled](./values.yaml#L385) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.slack.channels](./values.yaml#L389) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.slack.channels.default.name](./values.yaml#L392) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.slack.channels.default.notification.disabled](./values.yaml#L395) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.slack.channels.default.bindings.executors](./values.yaml#L398) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.slack.channels.default.bindings.sources](./values.yaml#L401) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.slack.token](./values.yaml#L405) | string | `""` | Slack token. |
| [communications.default-group.slack.notification.type](./values.yaml#L408) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.socketSlack.enabled](./values.yaml#L413) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.socketSlack.channels](./values.yaml#L417) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.socketSlack.channels.default.name](./values.yaml#L420) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.socketSlack.channels.default.bindings.executors](./values.yaml#L423) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.socketSlack.channels.default.bindings.sources](./values.yaml#L426) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.socketSlack.botToken](./values.yaml#L431) | string | `""` | Slack bot token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.appToken](./values.yaml#L434) | string | `""` | Slack app-level token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.notification.type](./values.yaml#L437) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.mattermost.enabled](./values.yaml#L441) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L443) | string | `"Botkube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L445) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L447) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by Botkube user. |
| [communications.default-group.mattermost.team](./values.yaml#L449) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where Botkube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L453) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"MATTERMOST_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L457) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.mattermost.channels.default.notification.disabled](./values.yaml#L460) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.mattermost.channels.default.bindings.executors](./values.yaml#L463) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.mattermost.channels.default.bindings.sources](./values.yaml#L466) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.mattermost.notification.type](./values.yaml#L471) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.teams.enabled](./values.yaml#L476) | bool | `false` | If true, enables MS Teams bot. |
| [communications.default-group.teams.botName](./values.yaml#L478) | string | `"Botkube"` | The Bot name set while registering Bot to MS Teams. |
| [communications.default-group.teams.appID](./values.yaml#L480) | string | `"APPLICATION_ID"` | The Botkube application ID generated while registering Bot to MS Teams. |
| [communications.default-group.teams.appPassword](./values.yaml#L482) | string | `"APPLICATION_PASSWORD"` | The Botkube application password generated while registering Bot to MS Teams. |
| [communications.default-group.teams.bindings.executors](./values.yaml#L485) | list | `["kubectl-read-only"]` | Executor bindings apply to all MS Teams channels where Botkube has access to. |
| [communications.default-group.teams.bindings.sources](./values.yaml#L488) | list | `["k8s-err-events","k8s-recommendation-events"]` | Source bindings apply to all channels which have notification turned on with `@Botkube notifier start` command. |
| [communications.default-group.teams.messagePath](./values.yaml#L492) | string | `"/bots/teams"` | The path in endpoint URL provided while registering Botkube to MS Teams. |
| [communications.default-group.teams.port](./values.yaml#L494) | int | `3978` | The Service port for bot endpoint on Botkube container. |
| [communications.default-group.discord.enabled](./values.yaml#L499) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L501) | string | `"DISCORD_TOKEN"` | Botkube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L503) | string | `"DISCORD_BOT_ID"` | Botkube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L507) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"id":"DISCORD_CHANNEL_ID","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L511) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.discord.channels.default.notification.disabled](./values.yaml#L514) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.discord.channels.default.bindings.executors](./values.yaml#L517) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.discord.channels.default.bindings.sources](./values.yaml#L520) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.discord.notification.type](./values.yaml#L525) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L530) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L534) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L536) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L538) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L540) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L542) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L544) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L547) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.indices](./values.yaml#L551) | object | `{"default":{"bindings":{"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L554) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.elasticsearch.indices.default.bindings.sources](./values.yaml#L560) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given index. |
| [communications.default-group.webhook.enabled](./values.yaml#L567) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L569) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [communications.default-group.webhook.bindings.sources](./values.yaml#L572) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for the webhook. |
| [settings.clusterName](./values.yaml#L579) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.lifecycleServer](./values.yaml#L582) | object | `{"enabled":true,"port":2113}` | Server configuration which exposes functionality related to the app lifecycle. |
| [settings.upgradeNotifier](./values.yaml#L586) | bool | `true` | If true, notifies about new Botkube releases. |
| [settings.log.level](./values.yaml#L590) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L592) | bool | `false` | If true, disable ANSI colors in logging. |
| [settings.systemConfigMap](./values.yaml#L595) | object | `{"name":"botkube-system"}` | Botkube's system ConfigMap where internal data is stored. |
| [settings.persistentConfig](./values.yaml#L600) | object | `{"runtime":{"configMap":{"annotations":{},"name":"botkube-runtime-config"},"fileName":"_runtime_state.yaml"},"startup":{"configMap":{"annotations":{},"name":"botkube-startup-config"},"fileName":"_startup_state.yaml"}}` | Persistent config contains ConfigMap where persisted configuration is stored. The persistent configuration is evaluated from both chart upgrade and Botkube commands used in runtime. |
| [ssl.enabled](./values.yaml#L615) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L621) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L624) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L627) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L634) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L645) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L655) | object | `{}` | Extra annotations to pass to the Botkube Deployment. |
| [extraAnnotations](./values.yaml#L662) | object | `{}` | Extra annotations to pass to the Botkube Pod. |
| [extraLabels](./values.yaml#L664) | object | `{}` | Extra labels to pass to the Botkube Pod. |
| [priorityClassName](./values.yaml#L666) | string | `""` | Priority class name for the Botkube Pod. |
| [nameOverride](./values.yaml#L669) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L671) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L677) | object | `{}` | The Botkube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L689) | list | `[]` | Extra environment variables to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L701) | list | `[]` | Extra volumes to pass to the Botkube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L716) | list | `[]` | Extra volume mounts to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L734) | object | `{}` | Node labels for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L738) | list | `[]` | Tolerations for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L742) | object | `{}` | Affinity for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L746) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for Botkube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L755) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L758) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L760) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L763) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L791) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://botkube.io/privacy#privacy-policy). |
| [configWatcher.enabled](./values.yaml#L796) | bool | `true` | If true, restarts the Botkube Pod on config changes. |
| [configWatcher.tmpDir](./values.yaml#L798) | string | `"/tmp/watched-cfg/"` | Directory, where watched configuration resources are stored. |
| [configWatcher.initialSyncTimeout](./values.yaml#L801) | int | `0` | Timeout for the initial Config Watcher sync. If set to 0, waiting for Config Watcher sync will be skipped. In a result, configuration changes may not reload Botkube app during the first few seconds after Botkube startup. |
| [configWatcher.image.registry](./values.yaml#L804) | string | `"ghcr.io"` | Config watcher image registry. |
| [configWatcher.image.repository](./values.yaml#L806) | string | `"kubeshop/k8s-sidecar"` | Config watcher image repository. |
| [configWatcher.image.tag](./values.yaml#L808) | string | `"ignore-initial-events"` | Config watcher image tag. |
| [configWatcher.image.pullPolicy](./values.yaml#L810) | string | `"IfNotPresent"` | Config watcher image pull policy. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine-grained access. This is useful if you are looking to run Botkube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the Botkube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding Botkube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "{role_arn_to_assume}"
```
