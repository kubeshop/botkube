# BotKube Use Case: Collaborative debugging Kubernetes resources via Slack

This examples showcase debugging failing Pod with network issue. You will learn:

- how BotKube notifies you about errors
- how to execute `kubectl` commands via `@Botkube`
- how the `kubectl` permission restriction works

## Prerequisites

Install the following applications:

- [k3d](https://k3d.io/v5.4.6/)
- [Helm](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)

### Set up local Kubernetes cluster

Run the command:

```bash
k3d cluster create svc-debug
```

### Deploy BotKube

1. Install [BotKube in your Slack workspace](https://botkube.io/installation/slack/#install-botkube-slack-app-to-your-slack-workspace).

2. Export required environment variables:

   ```bash
   export SLACK_BOT_TOKEN="{token}"
   export TEAM_SLACK_CHANNEL="{channel}" # e.g. gophers
   export ADMIN_SLACK_CHANNEL="{channel}" # e.g. admin
   ```

   > **Note**
   > Each channel need to exist and BotKube need to be invited.

3. Add BotKube Helm chart:

   ```bash
   helm repo add botkube https://charts.botkube.io
   helm repo update
   ```

4. Deploy BotKube:

   ```bash
   helm install botkube --version v0.13.0 --namespace botkube --create-namespace \
   -f ./examples/service-debugging/botkube-values.yaml \
   --set communications.default-group.slack.token=${SLACK_BOT_TOKEN} \
   --set communications.default-group.slack.channels.default.name=${TEAM_SLACK_CHANNEL} \
   --set communications.default-group.slack.channels.admin.name=${ADMIN_SLACK_CHANNEL} \
   --wait \
   botkube/botkube
   ```

### Deploy example app

Run the command:

```bash
kubectl apply -f ./examples/service-debugging/deploy
```

## Scenario

In this scenario, we will learn how to react to the error event sent on Slack channel.

1. You should see events sent by BotKube about created service:
2. After a minute, you should see an error event:

3. For now, we don't know too much about the error itself. To learn more, let's check the `meme` Pod logs:

   ```
   @Botkube logs -l app=meme
   ```

4. From the logs, we learnt that the `meme` Pod cannot call the `quote` Pod using defined Service URL. To be able to call the `quote` Pod we definitely need to have that Service defined. Let's see all Services:

   ```
   @Botkube get services
   ```


5. We can see that the `quote` Service is there. So we need to dig deeper into the configuration. We need to describe Service to check if there are any endpoints:

   ```
   @Botkube describe svc quote
   ```
	 Now it gets interesting: there are no endpoints, which means there isn't a single Pod that is matched by the Service selectors.

6. We need to check whether that the `quote` Pod is up and running:

   ```
   @Botube get pods
   ```
   💡 The `quote` Pod is up and running, so it might be a problem with incorrect labels.

7. There is a nice `--show-lables` flag which allows us to check that easily:

   ```
   @Botube get po quote-{} --show-labels
   ```

   We got it! The bug was found. The problem is with incorrect labels.

8. Add missing label to the quote Pod

   ```
   @Botube label pod quote-{} app=quote
   ```
	 We got an error. But that's yet another BotKube feature, which allows you to define executor permission per channel.

5. Let's Switch to `#admin` channel.

6. Add missing label to the `quote` Pod:

   ```
   @Botube label pod quote-{} app=quote
   ```

7. Restart the `meme` Pod:

   ```
   @Botkube delete po meme-{}
   ```

8. Run `logs` to confirm that `http://quote/quote` is reachable now:

   ```
   @Botkube logs meme-{}
   ```

### Summary

During the short demo, you can notice that:

- You don't need to install and configure any tools locally
- You don't need to repeat commands that were already executed by others
	- and you don't need to discover the same thing by your own when it was already discussed by the teammates
- You don't need to switch context - switch between Slack and your terminal
- You can define different `kubectl` permissions per channel

## Cleanup

Remove whole cluster:

```bash
k3d cluster delete svc-debug
```
