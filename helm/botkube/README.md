# Botkube

![Version: v9.99.9-dev](https://img.shields.io/badge/Version-v9.99.9--dev-informational?style=flat-square) ![AppVersion: v9.99.9-dev](https://img.shields.io/badge/AppVersion-v9.99.9--dev-informational?style=flat-square)

A virtual SRE, powered by AI.

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
| [rbac](./values.yaml#L43) | object | `{"create":true,"groups":{"botkube-plugins-default":{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}},"rules":[],"serviceAccountMountPath":"/var/run/7e7fd2f5-b15d-4803-bc52-f54fba357e76/secrets/kubernetes.io/serviceaccount","staticGroupName":""}` | Role Based Access for Botkube Pod and plugins. [Ref doc](https://kubernetes.io/docs/reference/access-authn-authz/rbac/ ). |
| [rbac.serviceAccountMountPath](./values.yaml#L47) | string | `"/var/run/7e7fd2f5-b15d-4803-bc52-f54fba357e76/secrets/kubernetes.io/serviceaccount"` | It is used to specify a custom path for mounting a service account to the Botkube deployment. This is important because we run plugins within the same Pod, and we want to avoid potential bugs when plugins rely on the default in-cluster K8s client configuration. Instead, they should always use kubeconfig specified directly for a given plugin. |
| [rbac.create](./values.yaml#L50) | bool | `true` | Configure RBAC resources for Botkube and (deprecated) `staticGroupName` subject with `rules`. For creating RBAC resources related to plugin permissions, use the `groups` property. |
| [rbac.rules](./values.yaml#L52) | list | `[]` | Deprecated. Use `rbac.groups` instead. |
| [rbac.staticGroupName](./values.yaml#L54) | string | `""` | Deprecated. Use `rbac.groups` instead. |
| [rbac.groups](./values.yaml#L56) | object | `{"botkube-plugins-default":{"create":true,"rules":[{"apiGroups":["*"],"resources":["*"],"verbs":["get","watch","list"]}]}}` | Use this to create RBAC resources for specified group subjects. |
| [kubeconfig.enabled](./values.yaml#L67) | bool | `false` | If true, enables overriding the Kubernetes auth. |
| [kubeconfig.base64Config](./values.yaml#L69) | string | `""` | A base64 encoded kubeconfig that will be stored in a Secret, mounted to the Pod, and specified in the KUBECONFIG environment variable. |
| [kubeconfig.existingSecret](./values.yaml#L74) | string | `""` | A Secret containing a kubeconfig to use.  |
| [actions](./values.yaml#L81) | object | See the `values.yaml` file for full object. | Map of actions. Action contains configuration for automation based on observed events. The property name under `actions` object is an alias for a given configuration. You can define multiple actions configuration with different names.   |
| [actions.describe-created-resource.enabled](./values.yaml#L84) | bool | `false` | If true, enables the action. |
| [actions.describe-created-resource.displayName](./values.yaml#L86) | string | `"Describe created resource"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.describe-created-resource.command](./values.yaml#L91) | string | See the `values.yaml` file for the command in the Go template form. | Command to execute when the action is triggered. You can use Go template (https://pkg.go.dev/text/template) together with all helper functions defined by Slim-Sprig library (https://go-task.github.io/slim-sprig). You can use the `{{ .Event }}` variable, which contains the event object that triggered the action. See all available Kubernetes event properties on https://github.com/kubeshop/botkube/blob/main/internal/source/kubernetes/event/event.go. |
| [actions.describe-created-resource.bindings](./values.yaml#L94) | object | `{"executors":["k8s-default-tools"],"sources":["k8s-create-events"]}` | Bindings for a given action. |
| [actions.describe-created-resource.bindings.sources](./values.yaml#L96) | list | `["k8s-create-events"]` | Event sources that trigger a given action. |
| [actions.describe-created-resource.bindings.executors](./values.yaml#L99) | list | `["k8s-default-tools"]` | Executors configuration used to execute a configured command. |
| [actions.show-logs-on-error.enabled](./values.yaml#L103) | bool | `false` | If true, enables the action. |
| [actions.show-logs-on-error.displayName](./values.yaml#L106) | string | `"Show logs on error"` | Action display name posted in the channels bound to the same source bindings. |
| [actions.show-logs-on-error.command](./values.yaml#L111) | string | See the `values.yaml` file for the command in the Go template form. | Command to execute when the action is triggered. You can use Go template (https://pkg.go.dev/text/template) together with all helper functions defined by Slim-Sprig library (https://go-task.github.io/slim-sprig). You can use the `{{ .Event }}` variable, which contains the event object that triggered the action. See all available Kubernetes event properties on https://github.com/kubeshop/botkube/blob/main/internal/source/kubernetes/event/event.go. |
| [actions.show-logs-on-error.bindings](./values.yaml#L113) | object | `{"executors":["k8s-default-tools"],"sources":["k8s-err-with-logs-events"]}` | Bindings for a given action. |
| [actions.show-logs-on-error.bindings.sources](./values.yaml#L115) | list | `["k8s-err-with-logs-events"]` | Event sources that trigger a given action. |
| [actions.show-logs-on-error.bindings.executors](./values.yaml#L118) | list | `["k8s-default-tools"]` | Executors configuration used to execute a configured command. |
| [sources](./values.yaml#L127) | object | See the `values.yaml` file for full object. | Map of sources. Source contains configuration for Kubernetes events and sending recommendations. The property name under `sources` object is an alias for a given configuration. You can define multiple sources configuration with different names. Key name is used as a binding reference.   |
| [sources.k8s-recommendation-events.botkube/kubernetes](./values.yaml#L132) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac](./values.yaml#L135) | object | `{"group":{"prefix":"","static":{"values":["botkube-plugins-default"]},"type":"Static"}}` | RBAC configuration for this plugin. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.type](./values.yaml#L138) | string | `"Static"` | Static impersonation for a given username and groups. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.prefix](./values.yaml#L140) | string | `""` | Prefix that will be applied to .static.value[*]. |
| [sources.k8s-recommendation-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-err-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-all-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-create-events.botkube/kubernetes.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [executors.k8s-default-tools.botkube/kubectl.context.rbac.group.static.values](./values.yaml#L143) | list | `["botkube-plugins-default"]` | Name of group.rbac.authorization.k8s.io the plugin will be bound to. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations](./values.yaml#L157) | object | `{"ingress":{"backendServiceValid":true,"tlsSecretValid":true},"pod":{"labelsSet":true,"noLatestImageTag":true}}` | Describes configuration for various recommendation insights. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod](./values.yaml#L159) | object | `{"labelsSet":true,"noLatestImageTag":true}` | Recommendations for Pod Kubernetes resource. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod.noLatestImageTag](./values.yaml#L161) | bool | `true` | If true, notifies about Pod containers that use `latest` tag for images. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.pod.labelsSet](./values.yaml#L163) | bool | `true` | If true, notifies about Pod resources created without labels. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress](./values.yaml#L165) | object | `{"backendServiceValid":true,"tlsSecretValid":true}` | Recommendations for Ingress Kubernetes resource. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress.backendServiceValid](./values.yaml#L167) | bool | `true` | If true, notifies about Ingress resources with invalid backend service reference. |
| [sources.k8s-recommendation-events.botkube/kubernetes.config.recommendations.ingress.tlsSecretValid](./values.yaml#L169) | bool | `true` | If true, notifies about Ingress resources with invalid TLS secret reference. |
| [sources.k8s-all-events.botkube/kubernetes](./values.yaml#L175) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters](./values.yaml#L181) | object | See the `values.yaml` file for full object. | Filter settings for various sources. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters.objectAnnotationChecker](./values.yaml#L183) | bool | `true` | If true, enables support for `botkube.io/disable` resource annotation. |
| [sources.k8s-all-events.botkube/kubernetes.config.filters.nodeEventsChecker](./values.yaml#L185) | bool | `true` | If true, filters out Node-related events that are not important. |
| [sources.k8s-all-events.botkube/kubernetes.config.namespaces](./values.yaml#L189) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-create-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L193) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-all-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L193) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-err-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L193) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.namespaces.include](./values.yaml#L193) | list | `[".*"]` | Include contains a list of allowed Namespaces. It can also contain regex expressions:  `- ".*"` - to specify all Namespaces. |
| [sources.k8s-all-events.botkube/kubernetes.config.event](./values.yaml#L203) | object | `{"message":{"exclude":[],"include":[]},"reason":{"exclude":[],"include":[]},"types":["create","delete","error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.types](./values.yaml#L205) | list | `["create","delete","error"]` | Lists all event types to be watched. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason](./values.yaml#L211) | object | `{"exclude":[],"include":[]}` | Optional list of exact values or regex patterns to filter events by event reason. Skipped, if both include/exclude lists are empty. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason.include](./values.yaml#L213) | list | `[]` | Include contains a list of allowed values. It can also contain regex expressions. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.reason.exclude](./values.yaml#L216) | list | `[]` | Exclude contains a list of values to be ignored even if allowed by Include. It can also contain regex expressions. Exclude list is checked before the Include list. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message](./values.yaml#L219) | object | `{"exclude":[],"include":[]}` | Optional list of exact values or regex patterns to filter event by event message. Skipped, if both include/exclude lists are empty. If a given event has multiple messages, it is considered a match if any of the messages match the constraints. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message.include](./values.yaml#L221) | list | `[]` | Include contains a list of allowed values. It can also contain regex expressions. |
| [sources.k8s-all-events.botkube/kubernetes.config.event.message.exclude](./values.yaml#L224) | list | `[]` | Exclude contains a list of values to be ignored even if allowed by Include. It can also contain regex expressions. Exclude list is checked before the Include list. |
| [sources.k8s-all-events.botkube/kubernetes.config.annotations](./values.yaml#L228) | object | `{}` | Filters Kubernetes resources to watch by annotations. Each resource needs to have all the specified annotations. Regex expressions are not supported. |
| [sources.k8s-all-events.botkube/kubernetes.config.labels](./values.yaml#L231) | object | `{}` | Filters Kubernetes resources to watch by labels. Each resource needs to have all the specified labels. Regex expressions are not supported. |
| [sources.k8s-all-events.botkube/kubernetes.config.resources](./values.yaml#L238) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources to watch. Resources are identified by its type in `{group}/{version}/{kind (plural)}` format. Examples: `apps/v1/deployments`, `v1/pods`. Each resource can override the namespaces and event configuration by using dedicated `event` and `namespaces` field. Also, each resource can specify its own `annotations`, `labels` and `name` regex. |
| [sources.k8s-err-events.botkube/kubernetes](./values.yaml#L352) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-err-events.botkube/kubernetes.config.namespaces](./values.yaml#L359) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-events.botkube/kubernetes.config.event](./values.yaml#L363) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-events.botkube/kubernetes.config.event.types](./values.yaml#L365) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-events.botkube/kubernetes.config.resources](./values.yaml#L370) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes](./values.yaml#L396) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.namespaces](./values.yaml#L403) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.event](./values.yaml#L407) | object | `{"types":["error"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.event.types](./values.yaml#L409) | list | `["error"]` | Lists all event types to be watched. |
| [sources.k8s-err-with-logs-events.botkube/kubernetes.config.resources](./values.yaml#L414) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [sources.k8s-create-events.botkube/kubernetes](./values.yaml#L427) | object | See the `values.yaml` file for full object. | Describes Kubernetes source configuration. |
| [sources.k8s-create-events.botkube/kubernetes.config.namespaces](./values.yaml#L434) | object | `{"include":[".*"]}` | Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object. |
| [sources.k8s-create-events.botkube/kubernetes.config.event](./values.yaml#L438) | object | `{"types":["create"]}` | Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the `resources` list, unless they are overridden by the resource's own `events` object. |
| [sources.k8s-create-events.botkube/kubernetes.config.event.types](./values.yaml#L440) | list | `["create"]` | Lists all event types to be watched. |
| [sources.k8s-create-events.botkube/kubernetes.config.resources](./values.yaml#L445) | list | See the `values.yaml` file for full object. | Describes the Kubernetes resources you want to watch. |
| [executors](./values.yaml#L463) | object | See the `values.yaml` file for full object. | Map of executors. Executor contains configuration for running `kubectl` commands. The property name under `executors` is an alias for a given configuration. You can define multiple executor configurations with different names. Key name is used as a binding reference.   |
| [executors.k8s-default-tools.botkube/kubectl.config](./values.yaml#L472) | object | See the `values.yaml` file for full object including optional properties related to interactive builder. | Custom kubectl configuration. |
| [aliases](./values.yaml#L497) | object | See the `values.yaml` file for full object. | Custom aliases for given commands. The aliases are replaced with the underlying command before executing it. Aliases can replace a single word or multiple ones. For example, you can define a `k` alias for `kubectl`, or `kgp` for `kubectl get pods`.   |
| [existingCommunicationsSecretName](./values.yaml#L518) | string | `""` | Configures existing Secret with communication settings. It MUST be in the `botkube` Namespace. To reload Botkube once it changes, add label `botkube.io/config-watch: "true"`.  |
| [communications](./values.yaml#L525) | object | See the `values.yaml` file for full object. | Map of communication groups. Communication group contains settings for multiple communication platforms. The property name under `communications` object is an alias for a given configuration group. You can define multiple communication groups with different names.   |
| [communications.default-group.socketSlack.enabled](./values.yaml#L530) | bool | `false` | If true, enables bot for Slack. |
| [communications.default-group.socketSlack.channels](./values.yaml#L534) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"SLACK_CHANNEL"}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.socketSlack.channels.default.name](./values.yaml#L537) | string | `"SLACK_CHANNEL"` | Slack channel name without '#' prefix where you have added Botkube and want to receive notifications in. |
| [communications.default-group.socketSlack.channels.default.bindings.executors](./values.yaml#L540) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.socketSlack.channels.default.bindings.sources](./values.yaml#L543) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.socketSlack.botToken](./values.yaml#L548) | string | `""` | Bot token for your own app for Slack. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.socketSlack.appToken](./values.yaml#L551) | string | `""` | App-level token for your own app for Slack. [Ref doc](https://api.slack.com/authentication/token-types). |
| [communications.default-group.mattermost.enabled](./values.yaml#L555) | bool | `false` | If true, enables Mattermost bot. |
| [communications.default-group.mattermost.botName](./values.yaml#L557) | string | `"Botkube"` | User in Mattermost which belongs the specified Personal Access token. |
| [communications.default-group.mattermost.url](./values.yaml#L559) | string | `"MATTERMOST_SERVER_URL"` | The URL (including http/https schema) where Mattermost is running. e.g https://example.com:9243 |
| [communications.default-group.mattermost.token](./values.yaml#L561) | string | `"MATTERMOST_TOKEN"` | Personal Access token generated by Botkube user. |
| [communications.default-group.mattermost.team](./values.yaml#L563) | string | `"MATTERMOST_TEAM"` | The Mattermost Team name where Botkube is added. |
| [communications.default-group.mattermost.channels](./values.yaml#L567) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"MATTERMOST_CHANNEL","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.mattermost.channels.default.name](./values.yaml#L571) | string | `"MATTERMOST_CHANNEL"` | The Mattermost channel name for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.mattermost.channels.default.notification.disabled](./values.yaml#L574) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.mattermost.channels.default.bindings.executors](./values.yaml#L577) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.mattermost.channels.default.bindings.sources](./values.yaml#L580) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.discord.enabled](./values.yaml#L587) | bool | `false` | If true, enables Discord bot. |
| [communications.default-group.discord.token](./values.yaml#L589) | string | `"DISCORD_TOKEN"` | Botkube Bot Token. |
| [communications.default-group.discord.botID](./values.yaml#L591) | string | `"DISCORD_BOT_ID"` | Botkube Application Client ID. |
| [communications.default-group.discord.channels](./values.yaml#L595) | object | `{"default":{"bindings":{"executors":["k8s-default-tools"],"sources":["k8s-err-events","k8s-recommendation-events"]},"id":"DISCORD_CHANNEL_ID","notification":{"disabled":false}}}` | Map of configured channels. The property name under `channels` object is an alias for a given configuration.   |
| [communications.default-group.discord.channels.default.id](./values.yaml#L599) | string | `"DISCORD_CHANNEL_ID"` | Discord channel ID for receiving Botkube alerts. The Botkube user needs to be added to it. |
| [communications.default-group.discord.channels.default.notification.disabled](./values.yaml#L602) | bool | `false` | If true, the notifications are not sent to the channel. They can be enabled with `@Botkube` command anytime. |
| [communications.default-group.discord.channels.default.bindings.executors](./values.yaml#L605) | list | `["k8s-default-tools"]` | Executors configuration for a given channel. |
| [communications.default-group.discord.channels.default.bindings.sources](./values.yaml#L608) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given channel. |
| [communications.default-group.elasticsearch.enabled](./values.yaml#L615) | bool | `false` | If true, enables Elasticsearch. |
| [communications.default-group.elasticsearch.awsSigning.enabled](./values.yaml#L619) | bool | `false` | If true, enables awsSigning using IAM for Elasticsearch hosted on AWS. Make sure AWS environment variables are set. [Ref doc](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html). |
| [communications.default-group.elasticsearch.awsSigning.awsRegion](./values.yaml#L621) | string | `"us-east-1"` | AWS region where Elasticsearch is deployed. |
| [communications.default-group.elasticsearch.awsSigning.roleArn](./values.yaml#L623) | string | `""` | AWS IAM Role arn to assume for credentials, use this only if you don't want to use the EC2 instance role or not running on AWS instance. |
| [communications.default-group.elasticsearch.server](./values.yaml#L625) | string | `"ELASTICSEARCH_ADDRESS"` | The server URL, e.g https://example.com:9243 |
| [communications.default-group.elasticsearch.username](./values.yaml#L627) | string | `"ELASTICSEARCH_USERNAME"` | Basic Auth username. |
| [communications.default-group.elasticsearch.password](./values.yaml#L629) | string | `"ELASTICSEARCH_PASSWORD"` | Basic Auth password. |
| [communications.default-group.elasticsearch.skipTLSVerify](./values.yaml#L632) | bool | `false` | If true, skips the verification of TLS certificate of the Elastic nodes. It's useful for clusters with self-signed certificates. |
| [communications.default-group.elasticsearch.logLevel](./values.yaml#L639) | string | `""` | Specify the log level for Elasticsearch client. Leave empty to disable logging.  |
| [communications.default-group.elasticsearch.indices](./values.yaml#L644) | object | `{"default":{"bindings":{"sources":["k8s-err-events","k8s-recommendation-events"]},"name":"botkube","replicas":0,"shards":1,"type":"botkube-event"}}` | Map of configured indices. The `indices` property name is an alias for a given configuration.   |
| [communications.default-group.elasticsearch.indices.default.name](./values.yaml#L647) | string | `"botkube"` | Configures Elasticsearch index settings. |
| [communications.default-group.elasticsearch.indices.default.bindings.sources](./values.yaml#L653) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for a given index. |
| [communications.default-group.webhook.enabled](./values.yaml#L660) | bool | `false` | If true, enables Webhook. |
| [communications.default-group.webhook.url](./values.yaml#L662) | string | `"WEBHOOK_URL"` | The Webhook URL, e.g.: https://example.com:80 |
| [communications.default-group.webhook.bindings.sources](./values.yaml#L665) | list | `["k8s-err-events","k8s-recommendation-events"]` | Notification sources configuration for the webhook. |
| [settings.clusterName](./values.yaml#L672) | string | `"not-configured"` | Cluster name to differentiate incoming messages. |
| [settings.healthPort](./values.yaml#L675) | int | `2114` | Health check port. |
| [settings.upgradeNotifier](./values.yaml#L677) | bool | `true` | If true, notifies about new Botkube releases. |
| [settings.log.level](./values.yaml#L681) | string | `"info"` | Sets one of the log levels. Allowed values: `info`, `warn`, `debug`, `error`, `fatal`, `panic`. |
| [settings.log.disableColors](./values.yaml#L683) | bool | `false` | If true, disable ANSI colors in logging. Ignored when `json` formatter is used. |
| [settings.log.formatter](./values.yaml#L685) | string | `"json"` | Configures log format. Allowed values: `text`, `json`. |
| [settings.systemConfigMap](./values.yaml#L688) | object | `{"name":"botkube-system"}` | Botkube's system ConfigMap where internal data is stored. |
| [settings.persistentConfig](./values.yaml#L693) | object | `{"runtime":{"configMap":{"annotations":{},"name":"botkube-runtime-config"},"fileName":"_runtime_state.yaml"},"startup":{"configMap":{"annotations":{},"name":"botkube-startup-config"},"fileName":"_startup_state.yaml"}}` | Persistent config contains ConfigMap where persisted configuration is stored. The persistent configuration is evaluated from both chart upgrade and Botkube commands used in runtime. |
| [ssl.enabled](./values.yaml#L708) | bool | `false` | If true, specify cert path in `config.ssl.cert` property or K8s Secret in `config.ssl.existingSecretName`. |
| [ssl.existingSecretName](./values.yaml#L714) | string | `""` | Using existing SSL Secret. It MUST be in `botkube` Namespace.  |
| [ssl.cert](./values.yaml#L717) | string | `""` | SSL Certificate file e.g certs/my-cert.crt. |
| [service](./values.yaml#L720) | object | `{"name":"metrics","port":2112,"targetPort":2112}` | Configures Service settings for ServiceMonitor CR. |
| [serviceMonitor](./values.yaml#L727) | object | `{"enabled":false,"interval":"10s","labels":{},"path":"/metrics","port":"metrics"}` | Configures ServiceMonitor settings. [Ref doc](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor). |
| [deployment.annotations](./values.yaml#L737) | object | `{}` | Extra annotations to pass to the Botkube Deployment. |
| [deployment.livenessProbe](./values.yaml#L739) | object | `{"failureThreshold":35,"initialDelaySeconds":1,"periodSeconds":2,"successThreshold":1,"timeoutSeconds":1}` | Liveness probe. |
| [deployment.livenessProbe.initialDelaySeconds](./values.yaml#L741) | int | `1` | The liveness probe initial delay seconds. |
| [deployment.livenessProbe.periodSeconds](./values.yaml#L743) | int | `2` | The liveness probe period seconds. |
| [deployment.livenessProbe.timeoutSeconds](./values.yaml#L745) | int | `1` | The liveness probe timeout seconds. |
| [deployment.livenessProbe.failureThreshold](./values.yaml#L747) | int | `35` | The liveness probe failure threshold. |
| [deployment.livenessProbe.successThreshold](./values.yaml#L749) | int | `1` | The liveness probe success threshold. |
| [deployment.readinessProbe](./values.yaml#L752) | object | `{"failureThreshold":35,"initialDelaySeconds":1,"periodSeconds":2,"successThreshold":1,"timeoutSeconds":1}` | Readiness probe. |
| [deployment.readinessProbe.initialDelaySeconds](./values.yaml#L754) | int | `1` | The readiness probe initial delay seconds. |
| [deployment.readinessProbe.periodSeconds](./values.yaml#L756) | int | `2` | The readiness probe period seconds. |
| [deployment.readinessProbe.timeoutSeconds](./values.yaml#L758) | int | `1` | The readiness probe timeout seconds. |
| [deployment.readinessProbe.failureThreshold](./values.yaml#L760) | int | `35` | The readiness probe failure threshold. |
| [deployment.readinessProbe.successThreshold](./values.yaml#L762) | int | `1` | The readiness probe success threshold. |
| [extraAnnotations](./values.yaml#L769) | object | `{}` | Extra annotations to pass to the Botkube Pod. |
| [extraLabels](./values.yaml#L771) | object | `{}` | Extra labels to pass to the Botkube Pod. |
| [priorityClassName](./values.yaml#L773) | string | `""` | Priority class name for the Botkube Pod. |
| [nameOverride](./values.yaml#L776) | string | `""` | Fully override "botkube.name" template. |
| [fullnameOverride](./values.yaml#L778) | string | `""` | Fully override "botkube.fullname" template. |
| [resources](./values.yaml#L784) | object | `{}` | The Botkube Pod resource request and limits. We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. [Ref docs](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/) |
| [extraEnv](./values.yaml#L796) | list | `[{"name":"LOG_LEVEL_SOURCE_BOTKUBE_KUBERNETES","value":"debug"}]` | Extra environment variables to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables). |
| [extraVolumes](./values.yaml#L810) | list | `[]` | Extra volumes to pass to the Botkube container. Mount it later with extraVolumeMounts. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#Volume). |
| [extraVolumeMounts](./values.yaml#L825) | list | `[]` | Extra volume mounts to pass to the Botkube container. [Ref docs](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1). |
| [nodeSelector](./values.yaml#L843) | object | `{}` | Node labels for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/). |
| [tolerations](./values.yaml#L847) | list | `[]` | Tolerations for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/). |
| [affinity](./values.yaml#L851) | object | `{}` | Affinity for Botkube Pod assignment. [Ref doc](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity). |
| [serviceAccount.create](./values.yaml#L855) | bool | `true` | If true, a ServiceAccount is automatically created. |
| [serviceAccount.name](./values.yaml#L858) | string | `""` | The name of the service account to use. If not set, a name is generated using the fullname template. |
| [serviceAccount.annotations](./values.yaml#L860) | object | `{}` | Extra annotations for the ServiceAccount. |
| [extraObjects](./values.yaml#L863) | list | `[]` | Extra Kubernetes resources to create. Helm templating is allowed as it is evaluated before creating the resources. |
| [analytics.disable](./values.yaml#L890) | bool | `false` | If true, sending anonymous analytics is disabled. To learn what date we collect, see the [Privacy Policy](https://botkube.io/privacy-policy). |
| [configWatcher](./values.yaml#L894) | object | `{"enabled":true,"inCluster":{"informerResyncPeriod":"10m"}}` | Parameters for the Config Watcher component which reloads Botkube on ConfigMap changes. It restarts Botkube when configuration data change is detected. It watches ConfigMaps and/or Secrets with the `botkube.io/config-watch: "true"` label from the namespace where Botkube is installed. |
| [configWatcher.enabled](./values.yaml#L896) | bool | `true` | If true, restarts the Botkube Pod on config changes. |
| [configWatcher.inCluster](./values.yaml#L898) | object | `{"informerResyncPeriod":"10m"}` | In-cluster Config Watcher configuration. It is used when remote configuration is not provided. |
| [configWatcher.inCluster.informerResyncPeriod](./values.yaml#L900) | string | `"10m"` | Resync period for the Config Watcher informers. |
| [plugins](./values.yaml#L903) | object | `{"cacheDir":"/tmp","healthCheckInterval":"10s","incomingWebhook":{"enabled":true,"port":2115,"targetPort":2115},"repositories":{"botkube":{"url":"https://storage.googleapis.com/botkube-plugins-latest/plugins-index.yaml"}},"restartPolicy":{"threshold":10,"type":"DeactivatePlugin"}}` | Configuration for Botkube executors and sources plugins. |
| [plugins.cacheDir](./values.yaml#L905) | string | `"/tmp"` | Directory, where downloaded plugins are cached. |
| [plugins.repositories](./values.yaml#L907) | object | `{"botkube":{"url":"https://storage.googleapis.com/botkube-plugins-latest/plugins-index.yaml"}}` | List of plugins repositories. Each repository defines the URL and optional `headers` |
| [plugins.repositories.botkube](./values.yaml#L909) | object | `{"url":"https://storage.googleapis.com/botkube-plugins-latest/plugins-index.yaml"}` | This repository serves officially supported Botkube plugins. |
| [plugins.incomingWebhook](./values.yaml#L914) | object | `{"enabled":true,"port":2115,"targetPort":2115}` | Configure Incoming webhook for source plugins. |
| [plugins.restartPolicy](./values.yaml#L919) | object | `{"threshold":10,"type":"DeactivatePlugin"}` | Botkube Restart Policy on plugin failure. |
| [plugins.restartPolicy.type](./values.yaml#L921) | string | `"DeactivatePlugin"` | Restart policy type. Allowed values: "RestartAgent", "DeactivatePlugin". |
| [plugins.restartPolicy.threshold](./values.yaml#L923) | int | `10` | Number of restarts before policy takes into effect. |
| [config](./values.yaml#L927) | object | `{"provider":{"apiKey":"","endpoint":"https://api.botkube.io/graphql","identifier":""}}` | Configuration for synchronizing Botkube configuration. |
| [config.provider](./values.yaml#L929) | object | `{"apiKey":"","endpoint":"https://api.botkube.io/graphql","identifier":""}` | Base provider definition. |
| [config.provider.identifier](./values.yaml#L932) | string | `""` | Unique identifier for remote Botkube settings. If set to an empty string, Botkube won't fetch remote configuration. |
| [config.provider.endpoint](./values.yaml#L934) | string | `"https://api.botkube.io/graphql"` | Endpoint to fetch Botkube settings from. |
| [config.provider.apiKey](./values.yaml#L936) | string | `""` | Key passed as a `X-API-Key` header to the provider's endpoint. |

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine-grained access. This is useful if you are looking to run Botkube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the Botkube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding Botkube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "{role_arn_to_assume}"
```
