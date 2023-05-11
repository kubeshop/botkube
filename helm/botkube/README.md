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
| [rbac](./values.yaml#L43) | object | `{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}],"staticGroupName":"botkube-plugins-default"}` | Role Based Access for Botkube Pod. [Ref doc](https://kubernetes.io/docs/admin/authorization/rbac/). |
| [kubeconfig.enabled](./values.yaml#L54) | bool | `false` | If true, enables overriding the Kubernetes auth. |
| [kubeconfig.base64Config](./values.yaml#L56) | string | `""` | A base64 encoded kubeconfig that will be stored in a Secret, mounted to the Pod, and specified in the KUBECONFIG environment variable. |
| [kubeconfig.existingSecret](./values.yaml#L61) | string | `""` | A Secret containing a kubeconfig to use.  |
| [actions](./values.yaml#L68) | object | See the `values.yaml` file for full object. | Map of actions. Action contains configuration for automation based on observed events. The property name under `actions` object is an alias for a given configuration. You can define multiple actions configuration with different names.   |
| [actions.describe-created-resource.enabled](./values.yaml#L71) | bool | `false` | If true, enables the action. |
| [actions.describe-created-resource.displayName](./values.yaml#L73) | string | `"Describe created resource"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.describe-created-resource.command](./values.yaml#L78) | string | See the `values.yaml` file for the command in the Go template form. | Command to execute when the action is triggered. You can use Go template (https://pkg.go.dev/text/template) together with all helper functions defined by Slim-Sprig library (https://go-task.github.io/slim-sprig). You can use the `{{ .Event }}` variable, which contains the event object that triggered the action. See all available Kubernetes event properties on https://github.com/kubeshop/botkube/blob/main/internal/source/kubernetes/event/event.go. |
| [actions.describe-created-resource.bindings](./values.yaml#L81) | object | `{"executors":["k8s-default-tools"],"sources":["k8s-create-events"]}` | Bindings for a given action. |
| [actions.describe-created-resource.bindings.sources](./values.yaml#L83) | list | `["k8s-create-events"]` | Event sources that trigger a given action. |
| [actions.describe-created-resource.bindings.executors](./values.yaml#L86) | list | `["k8s-default-tools"]` | Executors configuration used to execute a configured command. |
| [actions.show-logs-on-error.enabled](./values.yaml#L90) | bool | `false` | If true, enables the action. |
| [actions.show-logs-on-error.displayName](./values.yaml#L93) | string | `"Show logs on error"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.show-logs-on-error.command](./values.yaml#L98) | string | See the `values.yaml` file for the command in the Go template form. | Command to execute when the action is triggered. You can use Go template (https://pkg.go.dev/text/template) together with all helper functions defined by Slim-Sprig library (https://go-task.github.io/slim-sprig). You can use the `{{ .Event }}` variable, which contains the event object that triggered the action. See all available Kubernetes event properties on https://github.com/kubeshop/botkube/blob/main/internal/source/kubernetes/event/event.go. |
| [actions.show-logs-on-error.bindings](./values.yaml#L100) | object | `{"executors":["k8s-default-tools"],"sources":["k8s-err-with-logs-events"]}` | Bindings for a given action. |
| [actions.show-logs-on-error.bindings.sources](./values.yaml#L102) | list | `["k8s-err-with-logs-events"]` | Event sources that trigger a given action. |
| [actions.show-logs-on-error.bindings.executors](./values.yaml#L105) | list | `["k8s-default-tools"]` | Executors configuration used to execute a configured command. |
| [sources](./values.yaml#L114) | object | See the `values.yaml` file for full object. | Map of sources. Source contains configuration for Kubernetes events and sending recommendations. The property name under `sources` object is an alias for a given configuration. You can define multiple sources configuration with different names. Key name is used as a binding reference.   |
| [sources.k8s-recommendation-events.botkube/kubernetes](./values.yaml#L119) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [executors.k8s-default-tools.botkube/helm.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac](./values.yaml#L123) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [executors.k8s-default-tools.botkube/helm.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L126) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [executors.k8s-default-tools.botkube/helm.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.prefix](./values.yaml#L128) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [executors.k8s-default-tools.botkube/helm.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.static.values](./values.yaml#L131) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations](./values.yaml#L145) | object | `{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}` | Describes configuration for various recommendation insights. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod](./values.yaml#L147) | object | `{"labelsSet":true,"noLatestImageTag":true}` | Recommendations for Pod Kubernetes resource. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod.noLatestImageTag](./values.yaml#L149) | bool | `true` | If true, notifies about Pod containers that use `latest` tag for images. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod.labelsSet](./values.yaml#L151) | bool | `true` | If true, notifies about Pod resources created without labels. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress](./values.yaml#L153) | object | `{"backendServiceValid":true,"tlsSecretValid":true}` | Recommendations for Ingress Kubernetes resource. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress.backendServiceValid](./values.yaml#L155) | bool | `true` | If true, notifies about Ingress resources with invalid backend service reference. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress.tlsSecretValid](./values.yaml#L157) | bool | `true` | If true, notifies about Ingress resources with invalid TLS secret reference. |
| [sources.k8s-all-events.botkube/kubernetes](./values.yaml#L163) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters](./values.yaml#L169) | object | See the `values.yaml` file for full object. | Filter settings for various sources. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters.objectAnnotationChecker](./values.yaml#L171) | bool | `true` | If true, enables support for `botkube.io/disable` resource annotation. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters.nodeEventsChecker](./values.yaml#L173) | bool | `true` | If true, filters out Node-related events that are not important. |
| [sources.k8s-all-events.botkube/kubernetes.config.namespaces](./values.yaml#L177) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L181) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-all-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L181) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-create-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L181) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L181) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-all-events.botkube/kubernetes.config.event](./values.yaml#L191) | object | `{"message":{"exclude":[],"include":[]},"reason":{"exclude":[],"include":[]},"types":["create","delete","error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.types](./values.yaml#L193) | list | `["create","delete","error"]` | Lists all event types to be watched. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason](./values.yaml#L199) | object | `{"exclude":[],"include":[]}` | Optional list of exact values or regex patterns to filter events by event reason. Skipped, if both include/exclude lists are empty. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason.include](./values.yaml#L201) | list | `[]` | Include contains a list of allowed values. It can also contain regex expressions. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason.exclude](./values.yaml#L204) | list | `[]` | Exclude contains a list of values to be ignored even if allowed by Include. It can also contain regex expressions. Exclude list is checked before the Include list. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message](./values.yaml#L207) | object | `{"exclude":[],"include":[]}` | Optional list of exact values or regex patterns to filter event by event message. Skipped, if both include/exclude lists are empty. If a given event has multiple messages, it is considered a match if any of the messages match the constraints. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message.include](./values.yaml#L209) | list | `[]` | Include contains a list of allowed values. It can also contain regex expressions. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message.exclude](./values.yaml#L212) | list | `[]` | Exclude contains a list of values to be ignored even if allowed by Include. It can also contain regex expressions. Exclude list is checked before the Include list. |
| [sources.k8s-all-events.botkube/kubernetes.config.annotations](./values.yaml#L216) | object | `{}` | Filters Kubernetes resources to watch by annotations. Each resource needs to have all the specified annotations. Regex expressions are not supported. |
| [sources.k8s-all-events.botkube/kubernetes.config.labels](./values.yaml#L219) | object | `{}` | Filters Kubernetes resources to watch by labels. Each resource needs to have all the specified labels. Regex expressions are not supported. |
| [sources.k8s-all-events.botkube/kubernetes.config.resources](./values.yaml#L226) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources to watch. Resources are identified by its type in `{group}/{version}/{kind (plural)}` format. Examples: `apps/v1/deployments`, `v1/pods`. Each resource can override the namespaces and event configuration by using dedicated `event` and `namespaces` field. Also, each resource can specify its own `annotations`, `labels` and `name` regex. |
| [sources.k8s-err-events.botkube/kubernetes](./values.yaml#L336) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-err-events.botkube/kubernetes.config.namespaces](./values.yaml#L343) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.botkube/kubernetes.config.event](./values.yaml#L347) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-events.botkube/kubernetes.config.event.types](./values.yaml#L349) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-events.botkube/kubernetes.config.resources](./values.yaml#L354) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes](./values.yaml#L376) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.namespaces](./values.yaml#L383) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.event](./values.yaml#L387) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.event.types](./values.yaml#L389) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.resources](./values.yaml#L394) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-create-events.botkube/kubernetes](./values.yaml#L407) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-create-events.botkube/kubernetes.config.namespaces](./values.yaml#L414) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-create-events.botkube/kubernetes.config.event](./values.yaml#L418) | object | `{"types":["create"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-create-events.botkube/kubernetes.config.event.types](./values.yaml#L420) | list | `["create"]` | Lists all event types to be watched. |
| [sources.k8s-create-events.botkube/kubernetes.config.resources](./values.yaml#L425) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.prometheus.botkube/prometheus.enabled](./values.yaml#L442) | bool | `false` | If true, enables `prometheus` source. |
| [sources.prometheus.botkube/prometheus.config.url](./values.yaml#L445) | string | `"http://localhost:9090"` | Prometheus endpoint without api version and resource. |
| [sources.prometheus.botkube/prometheus.config.ignoreOldAlerts](./values.yaml#L447) | bool | `true` | If set as true, Prometheus source plugin will not send alerts that is created before plugin start time. |
| [sources.prometheus.botkube/prometheus.config.alertStates](./values.yaml#L449) | list | `["firing","pending","inactive"]` | Only the alerts that have state provided in this config will be sent as notification. https://pkg.go.dev/github.com/prometheus/prometheus/rules#AlertState |
| [sources.prometheus.botkube/prometheus.config.log](./values.yaml#L451) | object | `{"level":"info"}` | Logging configuration |
| [sources.prometheus.botkube/prometheus.config.log.level](./values.yaml#L453) | string | `"info"` | Log level |
| [executors](./values.yaml#L461) | object | See the `values.yaml` file for full object. | Map of executors. Executor contains configuration for running `kubectl` commands. The property name under `executors` is an alias for a given configuration. You can define multiple executor configurations with different names. Key name is used as a binding reference.   |
| [executors.k8s-default-tools.botkube/helm.enabled](./values.yaml#L467) | bool | `false` | If true, enables `helm` commands execution. |
| [executors.k8s-default-tools.botkube/helm.config.helmDriver](./values.yaml#L472) | string | `"secret"` | Allowed values are configmap, secret, memory. |
| [executors.k8s-default-tools.botkube/helm.config.helmConfigDir](./values.yaml#L474) | string | `"/tmp/helm/"` | Location for storing Helm configuration. |
| [executors.k8s-default-tools.botkube/helm.config.helmCacheDir](./values.yaml#L476) | string | `"/tmp/helm/.cache"` | Location for storing cached files. Must be under the Helm config directory. |
| [executors.k8s-default-tools.botkube/kubectl.config](./values.yaml#L485) | object | See the `values.yaml` file for full object including optional properties related to interactive builder. | Custom kubectl configuration. |
| [aliases](./values.yaml#L510) | object | See the `values.yaml` file for full object. | Custom aliases for given commands. The aliases are replaced with the underlying command before executing it. Aliases can replace a single word or multiple ones. For example, you can define a `k` alias for `kubectl`, or `kgp` for `kubectl get pods`.   |
| [existingCommunicationsSecretName](./values.yaml#L530) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace. To reload Botkube once it changes, add label `botkube.io/config-watch: "true"`.  |
| [communications](./values.yaml#L537) | object | See the `values.yaml` file for full object. | Map of communication groups. Communication group contains settings for multiple communication platforms. The property name under `communications` object is an alias for a given configuration group. You can define multiple communication groups with different names.   |
| [communications.default-group.socketSlack.enabled](./values.yaml#L542) | bool | `false` | If true, enables Slack bot. |
| [communications.default-group.socketSlack.channels](./values.yaml#L546) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.socketSlack.channels.default.name](./values.yaml#L549) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.socketSlack.channels.default.bindings.executors](./values.yaml#L552) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.socketSlack.channels.default.bindings.sources](./values.yaml#L555) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.socketSlack.botToken](./values.yaml#L560) | string | `""` | Slack bot token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.appToken](./values.yaml#L563) | string | `""` | Slack app-level token for your own Slack app. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.mattermost.enabled](./values.yaml#L567) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L569) | string | `"Botkube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L571) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L573) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by Botkube user. |
| [communications.default-group.mattermost.team](./values.yaml#L575) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where Botkube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L579) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"MATTERMOST_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L583) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.mattermost.channels.default.notification.disabled](./values.yaml#L586) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.mattermost.channels.default.bindings.executors](./values.yaml#L589) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.mattermost.channels.default.bindings.sources](./values.yaml#L592) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.teams.enabled](./values.yaml#L599) | bool | `false` | If true, enables MS Teams bot. |
| [communications.default-group.teams.botName](./values.yaml#L601) | string | `"Botkube"` | The Bot name set while registering Bot to MS Teams. |
| [communications.default-group.teams.appID](./values.yaml#L603) | string | `"APPLICATION_ID"` | The Botkube application ID generated while registering Bot to MS Teams. |
| [communications.default-group.teams.appPassword](./values.yaml#L605) | string | `"APPLICATION_PASSWORD"` | The Botkube application password generated while registering Bot to MS Teams. |
| [communications.default-group.teams.bindings.executors](./values.yaml#L608) | list | `["k8s-default-tools"]` | Executor bindings apply to all MS Teams channels where Botkube has access to. |
| [communications.default-group.teams.bindings.sources](./values.yaml#L611) | list | `["k8s-err-events","k8s-recommendation-events"]` | Source bindings apply to all channels which have notification turned on with `@Botkube enable notifications` command. |
| [communications.default-group.teams.messagePath](./values.yaml#L615) | string | `"/bots/teams"` | The path in endpoint URL provided while registering Botkube to MS Teams. |
| [communications.default-group.teams.port](./values.yaml#L617) | int | `3978` | The Service port for bot endpoint on Botkube container. |
| [communications.default-group.discord.enabled](./values.yaml#L622) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L624) | string | `"DISCORD_TOKEN"` | Botkube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L626) | string | `"DISCORD_BOT_ID"` | Botkube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L630) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"id":"DISCORD_CHANNEL_ID","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L634) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.discord.channels.default.notification.disabled](./values.yaml#L637) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.discord.channels.default.bindings.executors](./values.yaml#L640) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.discord.channels.default.bindings.sources](./values.yaml#L643) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L650) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L654) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L656) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L658) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L660) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L662) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L664) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L667) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.indices](./values.yaml#L671) | object | `{"default":{"bindings":{"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L674) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.elasticsearch.indices.default.bindings.sources](./values.yaml#L680) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given index. |
| [communications.default-group.webhook.enabled](./values.yaml#L687) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L689) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [communications.default-group.webhook.bindings.sources](./values.yaml#L692) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for the webhook. |
| [communications.default-group.slack](./values.yaml#L702) | object | See the `values.yaml` file for full object. | Settings for deprecated Slack integration. **DEPRECATED:** Legacy Slack integration has been deprecated and removed from the Slack App Directory. Use `socketSlack` instead. Read more here: https://docs.botkube.io/installation/slack/   |
| [settings.clusterName](./values.yaml#L720) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.lifecycleServer](./values.yaml#L723) | object | `{"enabled":true,"port":2113}` | Server configuration which exposes functionality related to the app lifecycle. |
| [settings.healthPort](./values.yaml#L726) | int | `2114` |  |
| [settings.upgradeNotifier](./values.yaml#L728) | bool | `true` | If true, notifies about new Botkube releases. |
| [settings.log.level](./values.yaml#L732) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L734) | bool | `false` | If true, disable ANSI colors in logging. |
| [settings.systemConfigMap](./values.yaml#L737) | object | `{"name":"botkube-system"}` | Botkube's system ConfigMap where internal data is stored. |
| [settings.persistentConfig](./values.yaml#L742) | object | `{"runtime":{"configMap":{"annotations":{},"name":"botkube-runtime-config"},"fileName":"_runtime_state.yaml"},"startup":{"configMap":{"annotations":{},"name":"botkube-startup-config"},"fileName":"_startup_state.yaml"}}` | Persistent config contains ConfigMap where persisted configuration is stored. The persistent configuration is evaluated from both chart upgrade and Botkube commands used in runtime. |
| [ssl.enabled](./values.yaml#L757) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L763) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L766) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L769) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [ingress](./values.yaml#L776) | object | `{"annotations":{"kubernetes.io/ingress.class":"nginx"},"create":false,"host":"HOST","tls":{"enabled":false,"secretName":""}}` | Configures Ingress settings that exposes MS Teams endpoint. [Ref doc](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource). |
| [serviceMonitor](./values.yaml#L787) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L797) | object | `{}` | Extra annotations to pass to the Botkube Deployment. |
| [deployment.livenessProbe](./values.yaml#L799) | object | `{"failureThreshold":5,"initialDelaySeconds":1,"periodSeconds":15,"successThreshold":1,"timeoutSeconds":1}` | Liveness probe. |
| [deployment.livenessProbe.initialDelaySeconds](./values.yaml#L801) | int | `1` | The liveness probe initial delay seconds. |
| [deployment.livenessProbe.periodSeconds](./values.yaml#L803) | int | `15` | The liveness probe period seconds. |
| [deployment.livenessProbe.timeoutSeconds](./values.yaml#L805) | int | `1` | The liveness probe timeout seconds. |
| [deployment.livenessProbe.failureThreshold](./values.yaml#L807) | int | `5` | The liveness probe failure threshold. |
| [deployment.livenessProbe.successThreshold](./values.yaml#L809) | int | `1` | The liveness probe success threshold. |
| [deployment.readinessProbe](./values.yaml#L812) | object | `{"failureThreshold":5,"initialDelaySeconds":1,"periodSeconds":15,"successThreshold":1,"timeoutSeconds":1}` | Readiness probe. |
| [deployment.readinessProbe.initialDelaySeconds](./values.yaml#L814) | int | `1` | The readiness probe initial delay seconds. |
| [deployment.readinessProbe.periodSeconds](./values.yaml#L816) | int | `15` | The readiness probe period seconds. |
| [deployment.readinessProbe.timeoutSeconds](./values.yaml#L818) | int | `1` | The readiness probe timeout seconds. |
| [deployment.readinessProbe.failureThreshold](./values.yaml#L820) | int | `5` | The readiness probe failure threshold. |
| [deployment.readinessProbe.successThreshold](./values.yaml#L822) | int | `1` | The readiness probe success threshold. |
| [extraAnnotations](./values.yaml#L829) | object | `{}` | Extra annotations to pass to the Botkube Pod. |
| [extraLabels](./values.yaml#L831) | object | `{}` | Extra labels to pass to the Botkube Pod. |
| [priorityClassName](./values.yaml#L833) | string | `""` | Priority class name for the Botkube Pod. |
| [nameOverride](./values.yaml#L836) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L838) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L844) | object | `{}` | The Botkube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/user-guide/compute-resources/) |
| [extraEnv](./values.yaml#L856) | list | `[{"name":"LOG_LEVEL_SOURCE_BOTKUBE_KUBERNETES","value":"debug"}]` | Extra environment variables to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L870) | list | `[]` | Extra volumes to pass to the Botkube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L885) | list | `[]` | Extra volume mounts to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L903) | object | `{}` | Node labels for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/user-guide/node-selection/). |
| [tolerations](./values.yaml#L907) | list | `[]` | Tolerations for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L911) | object | `{}` | Affinity for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [serviceAccount.create](./values.yaml#L915) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L918) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L920) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L923) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L951) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see [Privacy Policy](https://docs.botkube.io/privacy#privacy-policy). |
| [configWatcher.enabled](./values.yaml#L956) | bool | `true` | If true, restarts the Botkube Pod on config changes. |
| [configWatcher.tmpDir](./values.yaml#L958) | string | `"/tmp/watched-cfg/"` | Directory, where watched configuration resources are stored. |
| [configWatcher.initialSyncTimeout](./values.yaml#L961) | int | `0` | Timeout for the initial Config Watcher sync. If set to 0, waiting for Config Watcher sync will be skipped. In a result, configuration changes may not reload Botkube app during the first few seconds after Botkube startup. |
| [configWatcher.image.registry](./values.yaml#L964) | string | `"ghcr.io"` | Config watcher image registry. |
| [configWatcher.image.repository](./values.yaml#L966) | string | `"kubeshop/k8s-sidecar"` | Config watcher image repository. |
| [configWatcher.image.tag](./values.yaml#L968) | string | `"ignore-initial-events"` | Config watcher image tag. |
| [configWatcher.image.pullPolicy](./values.yaml#L970) | string | `"IfNotPresent"` | Config watcher image pull policy. |
| [plugins](./values.yaml#L973) | object | `{"cacheDir":"/tmp","repositories":{"botkube":{"url":"https://github.com/kubeshop/botkube/releases/download/v9.99.9-dev/plugins-index.yaml"}}}` | Configuration for Botkube executors and sources plugins. |
| [plugins.cacheDir](./values.yaml#L975) | string | `"/tmp"` | Directory, where downloaded plugins are cached. |
| [plugins.repositories](./values.yaml#L977) | object | `{"botkube":{"url":"https://github.com/kubeshop/botkube/releases/download/v9.99.9-dev/plugins-index.yaml"}}` | List of plugins repositories. |
| [plugins.repositories.botkube](./values.yaml#L979) | object | `{"url":"https://github.com/kubeshop/botkube/releases/download/v9.99.9-dev/plugins-index.yaml"}` | This repository serves officially supported Botkube plugins. |
| [config](./values.yaml#L983) | object | `{"provider":{"apiKey":"","endpoint":"https://api.botkube.io/graphql","identifier":""}}` | Configuration for synchronizing Botkube configuration. |
| [config.provider](./values.yaml#L985) | object | `{"apiKey":"","endpoint":"https://api.botkube.io/graphql","identifier":""}` | Base provider definition. |
| [config.provider.identifier](./values.yaml#L988) | string | `""` | Unique identifier for remote Botkube settings. If set to an empty string, Botkube won't fetch remote configuration. |
| [config.provider.endpoint](./values.yaml#L990) | string | `"https://api.botkube.io/graphql"` | Endpoint to fetch Botkube settings from. |
| [config.provider.apiKey](./values.yaml#L992) | string | `""` | Key passed as a `X-API-Key` header to the provider's endpoint. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine-grained access. This is useful if you are looking to run Botkube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the Botkube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding Botkube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "{role_arn_to_assume}"
```
