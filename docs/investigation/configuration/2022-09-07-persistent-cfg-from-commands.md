# Persistent configuration from BotKube commands

Created on 2022-09-07 by Paweł Kosiec ([@pkosiec](https://github.com/pkosiec))

<!-- toc -->

- [Goal](#goal)
- [Current approach](#current-approach)
  * [Configuration persistence for BotKube commands](#configuration-persistence-for-botkube-commands)
  * [Reloading configuration](#reloading-configuration)
- [Single source of truth](#single-source-of-truth)
  * [Simple PoC](#simple-poc)
  * [Helm chart installation](#helm-chart-installation)
    + [Do external state change](#do-external-state-change)
    + [Upgrade Helm chart](#upgrade-helm-chart)
    + [Rollback](#rollback)
  * [Implementation](#implementation)
- [Setting configuration from BotKube commands](#setting-configuration-from-botkube-commands)
  * [1. Dedicated command for manual restart](#1-dedicated-command-for-manual-restart)
  * [2. Restart app every time and post an updated message](#2-restart-app-every-time-and-post-an-updated-message)
  * [3. Distinguish two types of commands and restart if necessary](#3-distinguish-two-types-of-commands-and-restart-if-necessary)
  * [Rejected: Reload config dynamically](#rejected-reload-config-dynamically)
- [Decision](#decision)

<!-- tocstop -->

## Goal

- Make configuration done with `@BotKube` commands persistent.
- Read configuration both from installation/upgrade config (ConfigMap) and also commands.

## Current approach

### Configuration persistence for BotKube commands

BotKube commands change runtime configuration and it is not persisted.

### Reloading configuration

Config watcher watches for ConfigMap changes and restarts the app if it was updated.

Before restart, BotKube messages:

> Looks like the configuration is updated for cluster 'not-configured'. I shall halt my watch till I read it.

and then:

> My watch has ended for cluster 'not-configured'!
Please send @BotKube notifier start to enable notification once BotKube comes online.

Once online, BotKube says:

> ...and now my watch begins for cluster 'not-configured'! :crossed_swords:

So there are 3 messages in total when reloading the configuration.

## Single source of truth

While I was considering different approaches, I wanted to make sure we have a single source of truth for our config - a ConfigMap. The idea is that every `@BotKube` command that saves some state (e.g. `@BotKube notifier stop`), writes the state to the same ConfigMap. If we're using ConfigMap, the concurrent write already resolved by Kubernetes itself (`resourceVersion`).

- As it is right now, BotKube always loads the configuration during it start from the files (including mounted ConfigMap).
- `@BotKube` commands results in Kubernetes API calls which modify ConfigMap with config.
- As the ConfigMap is mounted, Config Watcher detects the change.

To support Helm upgrade and `@BotKube` commands, I modified the Helm chart to make it work.

### Simple PoC

This scenario proves that the single source of truth works as expected. It imitates changes done by `@BotKube` commands and Helm upgrade.
While in this scenario I used Secret, same thing can be achieved with ConfigMap.

Patch the `communicationsecret.yaml` file.

```bash
cp docs/investigation/configuration/assets/communicationsecret.yaml ./helm/botkube/templates/communicationsecret.yaml
```

### Helm chart installation

Create the following file and install Helm chart.

- Mattermost `default` channel has `notifications.disabled=false`
- Slack `default` channel has `notifications.disabled=true`

```bash
cat > /tmp/values.yaml << ENDOFFILE
communications:
  'default-group':
    # Settings for Slack
    slack:
      enabled: false
      token: token
      channels:
        'default':
          name: botkube-test
          notifications:
            disabled: true
          bindings:
            executors:
              - kubectl-read-only
              - kubectl-read-only2
            sources:
              - k8s-events

    mattermost:
      enabled: false
      botName: botkube
      notiftype: short
      team: org
      token: token
      url: http://mattermost-team-edition.default:8065
      channels:
        'default':
          name: "general"
          notifications:
            disabled: false
          bindings:
            executors:
              - kubectl-read-only
            sources:
              - k8s-events
        'test':
          name: "test"
          bindings:
            executors:
              - kubectl-read-only
            sources:
              - k8s-events
      notification:
        type: short                             # Change notification type short/long you want to receive. Type is optional and default is short.
settings:
  clusterName: dev
executors:
  'kubectl-read-only':
    kubectl:
      enabled: true
analytics:
  disable: true
ENDOFFILE
helm install botkube ./helm/botkube --namespace botkube --create-namespace  -f /tmp/values.yaml
```

See the secret:
```bash
kubectl get secret -n botkube botkube-communication-secret -o go-template='{{ index .data "comm_config.yaml" | base64decode }}'
```

#### Do external state change

Let's imitate a change, which will be done by `@BotKube` commands.
Get secret `k get secret -n botkube botkube-communication-secret -oyaml` and change Slack `notifications.disabled=false` for Slack `default` channel, and add `notifications.disabled=true` for Teams.

Alternatively, trust me and apply the Secret update:

It changes `notifications.disabled=false` for Slack `default` channel, and adds `notifications.disabled=true` for Teams.

```bash
cat > /tmp/secret.yaml << ENDOFFILE
apiVersion: v1
data:
  comm_config.yaml: IyBDb21tdW5pY2F0aW9uIHNldHRpbmdzCmNvbW11bmljYXRpb25zOgogIGRlZmF1bHQtZ3JvdXA6CiAgICBkaXNjb3JkOgogICAgICBib3RJRDogRElTQ09SRF9CT1RfSUQKICAgICAgY2hhbm5lbHM6CiAgICAgICAgZGVmYXVsdDoKICAgICAgICAgIGJpbmRpbmdzOgogICAgICAgICAgICBleGVjdXRvcnM6CiAgICAgICAgICAgIC0ga3ViZWN0bC1yZWFkLW9ubHkKICAgICAgICAgICAgc291cmNlczoKICAgICAgICAgICAgLSBrOHMtZXZlbnRzCiAgICAgICAgICBpZDogRElTQ09SRF9DSEFOTkVMX0lECiAgICAgIGVuYWJsZWQ6IGZhbHNlCiAgICAgIG5vdGlmaWNhdGlvbjoKICAgICAgICB0eXBlOiBzaG9ydAogICAgICB0b2tlbjogRElTQ09SRF9UT0tFTgogICAgZWxhc3RpY3NlYXJjaDoKICAgICAgYXdzU2lnbmluZzoKICAgICAgICBhd3NSZWdpb246IHVzLWVhc3QtMQogICAgICAgIGVuYWJsZWQ6IGZhbHNlCiAgICAgICAgcm9sZUFybjogIiIKICAgICAgZW5hYmxlZDogZmFsc2UKICAgICAgaW5kaWNlczoKICAgICAgICBkZWZhdWx0OgogICAgICAgICAgYmluZGluZ3M6CiAgICAgICAgICAgIHNvdXJjZXM6CiAgICAgICAgICAgIC0gazhzLWV2ZW50cwogICAgICAgICAgbmFtZTogYm90a3ViZQogICAgICAgICAgcmVwbGljYXM6IDAKICAgICAgICAgIHNoYXJkczogMQogICAgICAgICAgdHlwZTogYm90a3ViZS1ldmVudAogICAgICBwYXNzd29yZDogRUxBU1RJQ1NFQVJDSF9QQVNTV09SRAogICAgICBzZXJ2ZXI6IEVMQVNUSUNTRUFSQ0hfQUREUkVTUwogICAgICBza2lwVExTVmVyaWZ5OiBmYWxzZQogICAgICB1c2VybmFtZTogRUxBU1RJQ1NFQVJDSF9VU0VSTkFNRQogICAgbWF0dGVybW9zdDoKICAgICAgYm90TmFtZTogYm90a3ViZQogICAgICBjaGFubmVsczoKICAgICAgICBkZWZhdWx0OgogICAgICAgICAgYmluZGluZ3M6CiAgICAgICAgICAgIGV4ZWN1dG9yczoKICAgICAgICAgICAgLSBrdWJlY3RsLXJlYWQtb25seQogICAgICAgICAgICBzb3VyY2VzOgogICAgICAgICAgICAtIGs4cy1ldmVudHMKICAgICAgICAgIG5hbWU6IGdlbmVyYWwKICAgICAgICAgIG5vdGlmaWNhdGlvbnM6CiAgICAgICAgICAgIGRpc2FibGVkOiBmYWxzZQogICAgICAgIHRlc3Q6CiAgICAgICAgICBiaW5kaW5nczoKICAgICAgICAgICAgZXhlY3V0b3JzOgogICAgICAgICAgICAtIGt1YmVjdGwtcmVhZC1vbmx5CiAgICAgICAgICAgIHNvdXJjZXM6CiAgICAgICAgICAgIC0gazhzLWV2ZW50cwogICAgICAgICAgbmFtZTogdGVzdAogICAgICBlbmFibGVkOiBmYWxzZQogICAgICBub3RpZmljYXRpb246CiAgICAgICAgdHlwZTogc2hvcnQKICAgICAgbm90aWZ0eXBlOiBzaG9ydAogICAgICB0ZWFtOiBvcmcKICAgICAgdG9rZW46IHRva2VuCiAgICAgIHVybDogaHR0cDovL21hdHRlcm1vc3QtdGVhbS1lZGl0aW9uLmRlZmF1bHQ6ODA2NQogICAgc2xhY2s6CiAgICAgIGNoYW5uZWxzOgogICAgICAgIGRlZmF1bHQ6CiAgICAgICAgICBiaW5kaW5nczoKICAgICAgICAgICAgZXhlY3V0b3JzOgogICAgICAgICAgICAtIGt1YmVjdGwtcmVhZC1vbmx5CiAgICAgICAgICAgIC0ga3ViZWN0bC1yZWFkLW9ubHkyCiAgICAgICAgICAgIHNvdXJjZXM6CiAgICAgICAgICAgIC0gazhzLWV2ZW50cwogICAgICAgICAgbmFtZTogYm90a3ViZS10ZXN0CiAgICAgICAgICBub3RpZmljYXRpb25zOgogICAgICAgICAgICBkaXNhYmxlZDogZmFsc2UKICAgICAgZW5hYmxlZDogZmFsc2UKICAgICAgbm90aWZpY2F0aW9uOgogICAgICAgIHR5cGU6IHNob3J0CiAgICAgIHRva2VuOiB0b2tlbgogICAgdGVhbXM6CiAgICAgIGFwcElEOiBBUFBMSUNBVElPTl9JRAogICAgICBhcHBQYXNzd29yZDogQVBQTElDQVRJT05fUEFTU1dPUkQKICAgICAgYmluZGluZ3M6CiAgICAgICAgZXhlY3V0b3JzOgogICAgICAgIC0ga3ViZWN0bC1yZWFkLW9ubHkKICAgICAgICBzb3VyY2VzOgogICAgICAgIC0gazhzLWV2ZW50cwogICAgICBib3ROYW1lOiBCb3RLdWJlCiAgICAgIGVuYWJsZWQ6IGZhbHNlCiAgICAgIG5vdGlmaWNhdGlvbnM6CiAgICAgICAgZGlzYWJsZWQ6IHRydWUKICAgICAgbWVzc2FnZVBhdGg6IC9ib3RzL3RlYW1zCiAgICAgIG5vdGlmaWNhdGlvbjoKICAgICAgICB0eXBlOiBzaG9ydAogICAgICBwb3J0OiAzOTc4CiAgICB3ZWJob29rOgogICAgICBiaW5kaW5nczoKICAgICAgICBzb3VyY2VzOgogICAgICAgIC0gazhzLWV2ZW50cwogICAgICBlbmFibGVkOiBmYWxzZQogICAgICB1cmw6IFdFQkhPT0tfVVJMCg==
kind: Secret
metadata:
  annotations:
    meta.helm.sh/release-name: botkube
    meta.helm.sh/release-namespace: botkube
  labels:
    app.kubernetes.io/instance: botkube
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: botkube
    helm.sh/chart: botkube-v0.13.0
  name: botkube-communication-secret
  namespace: botkube
type: Opaque
ENDOFFILE
kubectl apply -f /tmp/secret.yaml -n botkube
```

See the modified secret:
```bash
kubectl get secret -n botkube botkube-communication-secret -o go-template='{{ index .data "comm_config.yaml" | base64decode }}'
```

#### Upgrade Helm chart

Create modified `values2.yaml` file. It doesn't specify notifications for Slack and Teams, but sets `notifications.disabled=true` for Mattermost `default` channel. Upgrade the release:

```bash
cat > /tmp/values2.yaml << ENDOFFILE
communications:
  'default-group':
    # Settings for Slack
    slack:
      enabled: false
      token: token
      channels:
        'default':
          name: botkube-test
          bindings:
            executors:
              - kubectl-read-only
              - kubectl-read-only2
            sources:
              - k8s-events

    mattermost:
      enabled: false
      botName: botkube
      notiftype: short
      team: org
      token: token
      url: http://mattermost-team-edition.default:8065
      channels:
        'default':
          name: "general"
          notifications:
            disabled: true
          bindings:
            executors:
              - kubectl-read-only
            sources:
              - k8s-events
        'test':
          name: "test"
          bindings:
            executors:
              - kubectl-read-only
            sources:
              - k8s-events
      notification:
        type: short                             # Change notification type short/long you want to receive. Type is optional and default is short.
settings:
  clusterName: dev
executors:
  'kubectl-read-only':
    kubectl:
      enabled: true
analytics:
  disable: true
ENDOFFILE
helm upgrade botkube ./helm/botkube --namespace botkube -f /tmp/values2.yaml
```

Get secret and see its value:

```bash
kubectl get secret -n botkube botkube-communication-secret -o go-template='{{ index .data "comm_config.yaml" | base64decode }}'
```

The result is:

- Slack `default` channel has `notifications.disabled=false` (from the external Secret change)
- Mattermost `default` channel has `notifications.disabled=true` (from the Helm upgrade)
- MS Teams has `notifications.disabled=true` (from the external Secret change)

The merge implementation works properly, and we can use it to support config set by both upgrade + commands.

#### Rollback

Do a rollback and get the secret:

```bash
helm rollback -n botkube botkube
kubectl get secret -n botkube botkube-communication-secret -o go-template='{{ index .data "comm_config.yaml" | base64decode }}'
```

Now the secret is restored to the state after Helm chart installation:
- Slack `default` channel has `notifications.disabled=true` (from the initial Helm chart installation)
- Mattermost `default` channel has `notifications.disabled=false` (from the initial Helm chart installation)
- MS Teams doesn't have `notifications.disabled` specified (like during initial Helm installation)

The external change is not taken into account when rolling back. While it could be perceived as a limitation, I don't think there's a problem with this behavior - we just need to document it well.

### Implementation

To make sure we know where to write the state with commands, and to enable support for custom `communication` (`existingCommunicationsSecretName` in values) secrets and other files, the state ConfigMap should be separate from all different configurations.

The state ConfigMap name and namespace is provided as a part of config.

```yaml
settings:
  state:
    configMap:
      name: "botkube-state"
      namespace: "botkube"
```

The ConfigMap data would look like this:

```yaml
kind: ConfigMap
metadata:
  name: botkube-state
  namespace: botkube
data:
  communications:
    'default-group':
      slack:
        channels:
          'default':
            notification:
              disabled: true # notifier start/stop for a given channel
            sources: # notification presets
              - foo
              - bar
  sources: # someday
    'default-group':
      kubernetes:
        filters:
          namespaceChecker:
            enabled: true
          objectAnnotationChecker:
            enabled: true
```

As we use `koanf` library for loading config, these values will be merged with other files and we'll still have the same single struct in the app as before.
The state ConfigMap is provided as the last item in loaded files, that's why it takes the precedence over other files.

However, in the `values.yaml` file we still are able to inline them.

```yaml
communications:
  'default-group':
    # Settings for Slack
    slack:
      enabled: false
      token: token
      notification: # existing property - later we can rename it to `notifications`
        type: short
      channels:
        'default':
          name: botkube-test
          notification: # later we can rename it to `notifications`
            disabled: true
          sources: # presets
            - foo 
            - bar
```

Initially, we could have duplication over these "state" properties, that is, have them both in communication Secret and "state" ConfigMap. However, as pointed before, the state ConfigMap will take precedence over the Secret.
Later, we could iterate over all nested elements and check whether it should be filtered from the e.g. Communication Secret or State ConfigMap. So we'd iterate over the full object twice (once in state ConfigMap, and second time in Communications secret).

## Setting configuration from BotKube commands

### 1. Dedicated command for manual restart

Once user changes the configuration with `@BotKube` commands, config watcher detects the changes. It doesn't quit BotKube, but just informs about config reload instead, (e.g. only the first time):

> "Configuration has been updated. Apply it with the `@BotKube reload` command."

The `@BotKube reload` is executed manually to apply new configuration, which would result in the app restart.

We don't need to change the "goodbye" and "hello" messages.

Pros:
- Very easy to implement and maintain
- Allows to fully control the time BotKube app restarts by the user (batch multiple configuration changes)

Cons:
- UX is not great, as user needs to execute the command manually
- Current `notifier start/stop` commands won't apply the change instantly, but the reload will be needed

### 2. Restart app every time and post an updated message

Every command, even `notifier start/stop` restarts the app. But we don't post "config change detected", "goodbye" and "hello" messages.

Instead, we post a notification when BotKube is online: `BotKube configuration for cluster "dev" has been reloaded 👍`.

How to implement it? There would be another "state" BotKube ConfigMap, which is not monitored by Config Watcher, but still loaded during BotKube startup. Let's call it "startup-state" for now.

When detecting new configuration, Config Watcher writes to the "startup-state":

```yaml
lastExitReason: "ConfigReloaded"
```

When BotKube launches again, it reads the config and posts `BotKube configuration for cluster "dev" has been reloaded 👍`. Then, it modifies the "startup-state" ConfigMap and removes the `lastExitReason` field, to not post the same message next time.

Initially, it is posted for all channels as it is right now. Later, we can include the channel name in the state and post it only to the channel where the command was executed.

Pros:
- Quite easy to implement and maintain
- BotKube reload is "hidden"

Cons:
- Current `notifier start/stop` commands will also reload BotKube. That shouldn't make big difference though, as it would be "hidden".

### 3. Distinguish two types of commands and restart if necessary

This is a variation of the previous approach. We could distinguish two types of commands:
- commands that don't restart BotKube, e.g. `notifier start/stop`
    - they save the state in runtime config and another ConfigMap, which is not monitored by Config Watcher, but still loaded during BotKube startup.
- commands that restart BotKube instantly, such as commands for notification presets
    - they post "Reloading configuration..." message and restart the app.

Initially, the "goodbye" and "welcome" messages could be kept as they are. Later we can combine it with option 2.

Pros:
- Quite easy to implement and maintain
- Current `notifier start/stop` still work instantly

Cons:
- If we want to combine it with option 2 approach to hide BotKube restart, it would be a bit more time-consuming to implement.

### Rejected: Reload config dynamically

This doesn't change much as basically it would be restarting all components without the app itself; there would be still the same issues as with current approach. Kubernetes spawns a new Pod instantly, so there shouldn't be much time difference.
It would complicate the code though (as we would need to have another level of abstraction to watch over full app and ensure it's not finished, but restarted instead).

Doing a diff and restarting just some updated components (e.g. just Slack Bot or whole source router) doesn't sound like something we'd like to implement and maintain, as it'll bring too much complexity into our code, and, in a result, unpredictability in the behavior.

## Decision

After team discussion (@ezodude, @huseyinbabal, @mszostok) we agree as follows:
- We'll choose the option no 3.
    - In the first implementation every BotKube restart related to configuration change will post "goodbye" and "hello" messages. This will be implemented as a part of [#704](https://github.com/kubeshop/botkube/issues/704).
    - If we have time as a part of this task, we will also implement the scope of 2:
        - Modify ConfigWatcher - save config
        - Read config in controller and post custom message (`BotKube configuration for cluster "dev" has been reloaded 👍`)
        - Clear welcome message after posting it
- We may still implement the `@BotKube reload` command from the [1. Dedicated command for manual restart](#1-dedicated-command-for-manual-restart) section later, to support operation use cases (e.g. update configuration during maintenance window). This will be defined as a follow-up task to see how big the community demand is.
