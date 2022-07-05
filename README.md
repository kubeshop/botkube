# BotKube Helm Charts

For complete documentation visit [https://botkube.io](https://botkube.io)

BotKube integration with [Slack](https://slack.com), [Mattermost](https://mattermost.com) or [Microsoft Teams](https://www.microsoft.com/microsoft-365/microsoft-teams/group-chat-software) helps you monitor your Kubernetes cluster, debug critical deployments and gives recommendations for standard practices by running checks on the Kubernetes resources.
You can also ask BotKube to execute kubectl commands on k8s cluster which helps debugging an application or cluster.

## Usage

[Helm](https://helm.sh) must be installed to use the charts. Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repo as follows:

```console
helm repo add botkube https://charts.botkube.io
```

You can then run `helm search repo botkube` to see the charts.

## Contributing

The source code of all [BotKube](https://botkube.io) Helm charts can be found in [github.com/kubeshop/botkube/helm/botkube](https://github.com/kubeshop/botkube/tree/main/helm/botkube).

We'd love to have you contribute! Please refer to our [contribution guidelines](https://botkube.io/contribute/) for details.

## License

[MIT License](https://botkube.io/license/).
