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
| [actions.describe-created-resource.enabled](./values.yaml#L61) | bool | `false` | If true, enables the action. |
| [actions.describe-created-resource.displayName](./values.yaml#L63) | string | `"Describe created resource"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.describe-created-resource.command](./values.yaml#L66) | string | `"kubectl describe {{ .Event.TypeMeta.Kind | lower }}{{ if .Event.Namespace }} -n {{ .Event.Namespace }}{{ end }} {{ .Event.Name }}"` | A text value denoting the command run by this action, may contain even based templated values. The executor is inferred directly from the command, e.g. here we require a kubectl executor |
| [actions.describe-created-resource.bindings](./values.yaml#L69) | object | `{"executors":["kubectl-read-only"],"sources":["k8s-create-events"]}` | Bindings for a given action. |
| [actions.describe-created-resource.bindings.sources](./values.yaml#L71) | list | `["k8s-create-events"]` | Sources of events that trigger a given action. |
| [actions.describe-created-resource.bindings.executors](./values.yaml#L74) | list | `["kubectl-read-only"]` | Executors configuration for a given automation. |
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
| [sources.k8s-all-events.kubernetes](./values.yaml#L124) | object | `{"annotations":{},"event":{"message":"","reason":"","types":["create","delete","error"]},"labels":{},"namespaces":{"include":[".*"]},"resources":[{"type":"v1/pods"},{"type":"v1/services"},{"type":"networking.k8s.io/v1/ingresses"},{"type":"v1/nodes"},{"type":"v1/namespaces"},{"type":"v1/persistentvolumes"},{"type":"v1/persistentvolumeclaims"},{"type":"v1/configmaps"},{"type":"rbac.authorization.k8s.io/v1/roles"},{"type":"rbac.authorization.k8s.io/v1/rolebindings"},{"type":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"type":"rbac.authorization.k8s.io/v1/clusterroles"},{"event":{"types":["create","update","delete","error"]},"type":"apps/v1/daemonsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.numberReady"],"includeDiff":true}},{"event":{"types":["create","update","delete","error"]},"type":"batch/v1/jobs","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.conditions[*].type"],"includeDiff":true}},{"event":{"types":["create","update","delete","error"]},"type":"apps/v1/deployments","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.availableReplicas"],"includeDiff":true}},{"event":{"types":["create","update","delete","error"]},"type":"apps/v1/statefulsets","updateSetting":{"fields":["spec.template.spec.containers[*].image","status.readyReplicas"],"includeDiff":true}}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.kubernetes.namespaces](./values.yaml#L128) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-all-events.kubernetes.event](./values.yaml#L141) | object | `{"message":"","reason":"","types":["create","delete","error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-all-events.kubernetes.event.types](./values.yaml#L143) | list | `["create","delete","error"]` | Lists all event types to be watched. |
| [sources.k8s-all-events.kubernetes.event.reason](./values.yaml#L148) | string | `""` | Optional regex to filter events by event reason. |
| [sources.k8s-all-events.kubernetes.event.message](./values.yaml#L150) | string | `""` | Optional regex to filter events by message. If a given event has multiple messages, it is considered a match if any of the messages match the regex. |
| [sources.k8s-all-events.kubernetes.annotations](./values.yaml#L153) | object | `{}` | Filters Kubernetes resources to watch by annotations. |
| [sources.k8s-all-events.kubernetes.labels](./values.yaml#L155) | object | `{}` | Filters Kubernetes resources to watch by labels. |
| [sources.k8s-all-events.kubernetes.resources](./values.yaml#L162) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources to watch. Resources are identified by its type in `{group}/{version}/{kind (plural)}` format. Examples: `apps/v1/deployments`, `v1/pods`. Each resource can override the namespaces and event configuration by using dedicated `event` and `namespaces` field. Also, each resource can specify its own `annotations`, `labels` and `name` regex. |
| [sources.k8s-err-events.kubernetes](./values.yaml#L258) | object | `{"event":{"types":["error"]},"namespaces":{"include":[".*"]},"resources":[{"type":"v1/pods"},{"type":"v1/services"},{"type":"networking.k8s.io/v1/ingresses"},{"type":"v1/nodes"},{"type":"v1/namespaces"},{"type":"v1/persistentvolumes"},{"type":"v1/persistentvolumeclaims"},{"type":"v1/configmaps"},{"type":"rbac.authorization.k8s.io/v1/roles"},{"type":"rbac.authorization.k8s.io/v1/rolebindings"},{"type":"rbac.authorization.k8s.io/v1/clusterrolebindings"},{"type":"rbac.authorization.k8s.io/v1/clusterroles"},{"type":"apps/v1/deployments"},{"type":"apps/v1/statefulsets"},{"type":"apps/v1/daemonsets"},{"type":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-err-events.kubernetes.namespaces](./values.yaml#L262) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.kubernetes.event](./values.yaml#L266) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-events.kubernetes.event.types](./values.yaml#L268) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-events.kubernetes.resources](./values.yaml#L273) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-with-logs-events.kubernetes](./values.yaml#L294) | object | `{"event":{"types":["error"]},"namespaces":{"include":[".*"]},"resources":[{"type":"v1/pods"},{"type":"apps/v1/deployments"},{"type":"apps/v1/statefulsets"},{"type":"apps/v1/daemonsets"},{"type":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-err-with-logs-events.kubernetes.namespaces](./values.yaml#L298) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-with-logs-events.kubernetes.event](./values.yaml#L302) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-with-logs-events.kubernetes.event.types](./values.yaml#L304) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-with-logs-events.kubernetes.resources](./values.yaml#L309) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-create-events.kubernetes](./values.yaml#L321) | object | `{"event":{"types":["create"]},"namespaces":{"include":[".*"]},"resources":[{"type":"v1/pods"},{"type":"v1/services"},{"type":"networking.k8s.io/v1/ingresses"},{"type":"v1/nodes"},{"type":"v1/namespaces"},{"type":"v1/configmaps"},{"type":"apps/v1/deployments"},{"type":"apps/v1/statefulsets"},{"type":"apps/v1/daemonsets"},{"type":"batch/v1/jobs"}]}` | Describes Kubernetes source configuration. |
| [sources.k8s-create-events.kubernetes.namespaces](./values.yaml#L325) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-create-events.kubernetes.event](./values.yaml#L329) | object | `{"types":["create"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-create-events.kubernetes.event.types](./values.yaml#L331) | list | `["create"]` | Lists all event types to be watched. |
| [sources.k8s-create-events.kubernetes.resources](./values.yaml#L336) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [filters](./values.yaml#L352) | object | See the `values.yaml` file for full object. | Filter settings for various sources. Currently, all filters are globally enabled or disabled. You can enable or disable filters with `@Botkube filters` commands. |
| [filters.kubernetes.objectAnnotationChecker](./values.yaml#L355) | bool | `true` | If true, enables support for `botkube.io/disable` and `botkube.io/channel` resource annotations. |
| [filters.kubernetes.nodeEventsChecker](./values.yaml#L357) | bool | `true` | If true, filters out Node-related events that are not important. |
| [executors](./values.yaml#L365) | object | See the `values.yaml` file for full object. | Map of executors. Executor contains configuration for running `kubectl` commands. The property name under `executors` is an alias for a given configuration. You can define multiple executor configurations with different names. Key name is used as a binding reference.   |
| [executors.kubectl-read-only.kubectl.namespaces.include](./values.yaml#L373) | list | `[".*"]` | List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions:  `- ".*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.namespaces.exclude](./values.yaml#L378) | list | `[]` | List of ignored Kubernetes Namespace. It can also contain a regex expressions:  `- "test-.*"` - to specify all Namespaces. |
| [executors.kubectl-read-only.kubectl.enabled](./values.yaml#L380) | bool | `false` | If true, enables `kubectl` commands execution. |
| [executors.kubectl-read-only.kubectl.commands.verbs](./values.yaml#L384) | list | `["api-resources","api-versions","cluster-info","describe","explain","get","logs","top"]` | Configures which `kubectl` methods are allowed. |
| [executors.kubectl-read-only.kubectl.commands.resources](./values.yaml#L386) | list | `["deployments","pods","namespaces","daemonsets","statefulsets","storageclasses","nodes","configmaps","services","ingresses"]` | Configures which K8s resource are allowed. |
| [executors.kubectl-read-only.kubectl.defaultNamespace](./values.yaml#L388) | string | `"default"` | Configures the default Namespace for executing Botkube `kubectl` commands. If not set, uses the 'default'. |
| [executors.kubectl-read-only.kubectl.restrictAccess](./values.yaml#L390) | bool | `false` | If true, enables commands execution from configured channel only. |
| [existingCommunicationsSecretName](./values.yaml#L401) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace. To reload Botkube once it changes, add label `botkube.io/config-watch: "true"`.  |
| [communications](./values.yaml#L408) | object | See the `values.yaml` file for full object. | Map of communication groups. Communication group contains settings for multiple communication platforms. The property name under `communications` object is an alias for a given configuration group. You can define multiple communication groups with different names.   |
| [communications.default-group.slack.enabled](./values.yaml#L413) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.slack.channels](./values.yaml#L417) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.slack.channels.default.name](./values.yaml#L420) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.slack.channels.default.notification.disabled](./values.yaml#L423) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.slack.channels.default.bindings.executors](./values.yaml#L426) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.slack.channels.default.bindings.sources](./values.yaml#L429) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.slack.token](./values.yaml#L433) | string | `""` | Slack token. |
| [communications.default-group.slack.notification.type](./values.yaml#L436) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.socketSlack.enabled](./values.yaml#L441) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.socketSlack.channels](./values.yaml#L445) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.socketSlack.channels.default.name](./values.yaml#L448) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.socketSlack.channels.default.bindings.executors](./values.yaml#L451) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.socketSlack.channels.default.bindings.sources](./values.yaml#L454) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.socketSlack.botToken](./values.yaml#L459) | string | `""` | Slack bot token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.appToken](./values.yaml#L462) | string | `""` | Slack app-level token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.notification.type](./values.yaml#L465) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.mattermost.enabled](./values.yaml#L469) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L471) | string | `"Botkube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L473) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L475) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by Botkube user. |
| [communications.default-group.mattermost.team](./values.yaml#L477) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where Botkube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L481) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"MATTERMOST_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L485) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.mattermost.channels.default.notification.disabled](./values.yaml#L488) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.mattermost.channels.default.bindings.executors](./values.yaml#L491) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.mattermost.channels.default.bindings.sources](./values.yaml#L494) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.mattermost.notification.type](./values.yaml#L499) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.teams.enabled](./values.yaml#L504) | bool | `false` | If true, enables MS Teams bot. |
| [communications.default-group.teams.botName](./values.yaml#L506) | string | `"Botkube"` | The Bot name set while registering Bot to MS Teams. |
| [communications.default-group.teams.appID](./values.yaml#L508) | string | `"APPLICATION_ID"` | The Botkube application ID generated while registering Bot to MS Teams. |
| [communications.default-group.teams.appPassword](./values.yaml#L510) | string | `"APPLICATION_PASSWORD"` | The Botkube application password generated while registering Bot to MS Teams. |
| [communications.default-group.teams.bindings.executors](./values.yaml#L513) | list | `["kubectl-read-only"]` | Executor bindings apply to all MS Teams channels where Botkube has access to. |
| [communications.default-group.teams.bindings.sources](./values.yaml#L516) | list | `["k8s-err-events","k8s-recommendation-events"]` | Source bindings apply to all channels which have notification turned on with `@Botkube notifier start` command. |
| [communications.default-group.teams.messagePath](./values.yaml#L520) | string | `"/bots/teams"` | The path in endpoint URL provided while registering Botkube to MS Teams. |
| [communications.default-group.teams.port](./values.yaml#L522) | int | `3978` | The Service port for bot endpoint on Botkube container. |
| [communications.default-group.discord.enabled](./values.yaml#L527) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L529) | string | `"DISCORD_TOKEN"` | Botkube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L531) | string | `"DISCORD_BOT_ID"` | Botkube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L535) | object | `{"default":{"bindings":{"executors":["kubectl-read-only"],"sources":["k8s-err-events","k8s-recommendation-events"]},"id":"DISCORD_CHANNEL_ID","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L539) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.discord.channels.default.notification.disabled](./values.yaml#L542) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.discord.channels.default.bindings.executors](./values.yaml#L545) | list | `["kubectl-read-only"]` | Executors configuration for a given channel. |
| [communications.default-group.discord.channels.default.bindings.sources](./values.yaml#L548) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.discord.notification.type](./values.yaml#L553) | string | `"short"` | Configures notification type that are sent. Possible values: `short`, `long`. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L558) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L562) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L564) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L566) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L568) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L570) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L572) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L575) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.indices](./values.yaml#L579) | object | `{"default":{"bindings":{"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L582) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.elasticsearch.indices.default.bindings.sources](./values.yaml#L588) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given index. |
| [communications.default-group.webhook.enabled](./values.yaml#L595) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L597) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [communications.default-group.webhook.bindings.sources](./values.yaml#L600) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for the webhook. |
| [settings.clusterName](./values.yaml#L607) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.lifecycleServer](./values.yaml#L610) | object | `{"enabled":true,"port":2113}` | Server configuration which exposes functionality related to the app lifecycle. |
| [settings.upgradeNotifier](./values.yaml#L614) | bool | `true` | If true, notifies about new Botkube releases. |
| [settings.log.level](./values.yaml#L618) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L620) | bool | `false` | If true, disable ANSI colors in logging. |
| [settings.systemConfigMap](./values.yaml#L623) | object | `{"name":"botkube-system"}` | Botkube's system ConfigMap where internal data is stored. |
| [settings.persistentConfig](./values.yaml#L628) | object | `{"runtime":{"configMap":{"annotations":{},"name":"botkube-runtime-config"},"fileName":"_runtime_state.yaml"},"startup":{"configMap":{"annotations":{},"name":"botkube-startup-config"},"fileName":"_startup_state.yaml"}}` | Persistent config contains ConfigMap where persisted configuration is stored. The persistent configuration is evaluated from both chart upgrade and Botkube commands used in runtime. |
| [ssl.enabled](./values.yaml#L643) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L649) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L652) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L655) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L662) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L673) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L683) | object | `{}` | Extra annotations to pass to the Botkube Deployment. |
| [extraAnnotations](./values.yaml#L690) | object | `{}` | Extra annotations to pass to the Botkube Pod. |
| [extraLabels](./values.yaml#L692) | object | `{}` | Extra labels to pass to the Botkube Pod. |
| [priorityClassName](./values.yaml#L694) | string | `""` | Priority class name for the Botkube Pod. |
| [nameOverride](./values.yaml#L697) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L699) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L705) | object | `{}` | The Botkube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L717) | list | `[]` | Extra environment variables to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L729) | list | `[]` | Extra volumes to pass to the Botkube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L744) | list | `[]` | Extra volume mounts to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L762) | object | `{}` | Node labels for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L766) | list | `[]` | Tolerations for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L770) | object | `{}` | Affinity for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [rbac](./values.yaml#L774) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}` | Role Based Access for Botkube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [serviceAccount.create](./values.yaml#L783) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L786) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L788) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L791) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L819) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://botkube.io/privacy#privacy-policy). |
| [configWatcher.enabled](./values.yaml#L824) | bool | `true` | If true, restarts the Botkube Pod on config changes. |
| [configWatcher.tmpDir](./values.yaml#L826) | string | `"/tmp/watched-cfg/"` | Directory, where watched configuration resources are stored. |
| [configWatcher.initialSyncTimeout](./values.yaml#L829) | int | `0` | Timeout for the initial Config Watcher sync. If set to 0, waiting for Config Watcher sync will be skipped. In a result, configuration changes may not reload Botkube app during the first few seconds after Botkube startup. |
| [configWatcher.image.registry](./values.yaml#L832) | string | `"ghcr.io"` | Config watcher image registry. |
| [configWatcher.image.repository](./values.yaml#L834) | string | `"kubeshop/k8s-sidecar"` | Config watcher image repository. |
| [configWatcher.image.tag](./values.yaml#L836) | string | `"ignore-initial-events"` | Config watcher image tag. |
| [configWatcher.image.pullPolicy](./values.yaml#L838) | string | `"IfNotPresent"` | Config watcher image pull policy. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine-grained access. This is useful if you are looking to run Botkube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the Botkube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding Botkube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "{role_arn_to_assume}"
```
