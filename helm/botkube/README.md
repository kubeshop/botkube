# BotKube

![Version: v0.11.0](https://img.shields.io/badge/Version-v0.11.0-informational?style=flat-square) ![AppVersion: v0.11.0](https://img.shields.io/badge/AppVersion-v0.11.0-informational?style=flat-square)

Controller for the BotKube Slack app which helps you monitor your Kubernetes cluster, debug deployments and run specific checks on resources in the cluster.

**Homepage:** <https://www.botkube.io>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| PrasadG193 | prasad.ghangal@gmail.com |  |
| ssudake21 | sanket@infracloud.io |  |

## Source Code

* <https://github.com/infracloudio/botkube>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| communications.elasticsearch.awsSigning.awsRegion | string | `"us-east-1"` |  |
| communications.elasticsearch.awsSigning.enabled | bool | `false` |  |
| communications.elasticsearch.awsSigning.roleArn | string | `""` |  |
| communications.elasticsearch.enabled | bool | `false` |  |
| communications.elasticsearch.index.name | string | `"botkube"` |  |
| communications.elasticsearch.index.replicas | int | `0` |  |
| communications.elasticsearch.index.shards | int | `1` |  |
| communications.elasticsearch.index.type | string | `"botkube-event"` |  |
| communications.elasticsearch.password | string | `"ELASTICSEARCH_PASSWORD"` |  |
| communications.elasticsearch.server | string | `"ELASTICSEARCH_ADDRESS"` |  |
| communications.elasticsearch.username | string | `"ELASTICSEARCH_USERNAME"` |  |
| communications.mattermost.channel | string | `"MATTERMOST_CHANNEL"` |  |
| communications.mattermost.enabled | bool | `false` |  |
| communications.mattermost.notiftype | string | `"short"` |  |
| communications.mattermost.team | string | `"MATTERMOST_TEAM"` |  |
| communications.mattermost.token | string | `"MATTERMOST_TOKEN"` |  |
| communications.mattermost.url | string | `"MATTERMOST_SERVER_URL"` |  |
| communications.slack.channel | string | `"SLACK_CHANNEL"` |  |
| communications.slack.enabled | bool | `false` |  |
| communications.slack.notiftype | string | `"short"` |  |
| communications.slack.token | string | `"SLACK_API_TOKEN"` |  |
| communications.teams.appID | string | `"APPLICATION_ID"` |  |
| communications.teams.appPassword | string | `"APPLICATION_PASSWORD"` |  |
| communications.teams.enabled | bool | `false` |  |
| communications.teams.notiftype | string | `"short"` |  |
| communications.teams.port | int | `3978` |  |
| communications.webhook.enabled | bool | `false` |  |
| communications.webhook.url | string | `"WEBHOOK_URL"` |  |
| config.recommendations | bool | `true` |  |
| config.resources | list | [] | |
| config.settings.clustername | string | `"not-configured"` |  |
| config.settings.configwatcher | bool | `true` |  |
| config.settings.kubectl.commands.resources | list | [] |  |
| config.settings.kubectl.commands.verbs | list | [] |  |
| config.settings.kubectl.defaultNamespace | string | `"default"` |  |
| config.settings.kubectl.enabled | bool | `false` |  |
| config.settings.kubectl.restrictAccess | bool | `false` |  |
| config.settings.upgradeNotifier | bool | `true` |  |
| config.ssl.enabled | bool | `false` |  |
| extraAnnotations | object | `{}` |  |
| extraEnv | string | `nil` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"infracloudio/botkube"` |  |
| image.tag | string | `"latest"` |  |
| ingress.annotations."kubernetes.io/ingress.class" | string | `"nginx"` |  |
| ingress.create | bool | `false` |  |
| ingress.host | string | `"HOST"` |  |
| ingress.tls.enabled | bool | `false` |  |
| ingress.tls.secretName | string | `""` |  |
| ingress.urlPath | string | `"/"` |  |
| logLevel | string | `"info"` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podSecurityPolicy.enabled | bool | `false` |  |
| priorityClassName | string | `""` |  |
| rbac.create | bool | `true` |  |
| rbac.rules | list | [] |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext.runAsGroup | int | `101` |  |
| securityContext.runAsUser | int | `101` |  |
| service.name | string | `"metrics"` |  |
| service.port | int | `2112` |  |
| service.targetPort | int | `2112` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceMonitor.enabled | bool | `false` |  |
| serviceMonitor.interval | string | `"10s"` |  |
| serviceMonitor.labels | object | `{}` |  |
| serviceMonitor.path | string | `"/metrics"` |  |
| serviceMonitor.port | string | `"metrics"` |  |
| tolerations | list | `[]` |  |
