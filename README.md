# BotKube Helm Charts

BotKube helps you monitor your Kubernetes cluster, debug critical deployments and gives recommendations for standard practices by running checks on the Kubernetes resources. It integrates with multiple communication platforms, such as [Slack](https://slack.com) or [Mattermost](https://mattermost.com).

You can also execute `kubectl` commands on K8s cluster via BotKube which helps debugging an application or cluster.

For complete documentation visit [https://botkube.io](https://botkube.io).

## Usage

[Helm](https://helm.sh) must be installed to use the charts. Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repo as follows:

```console
helm repo add botkube https://charts.botkube.io
```

You can then run `helm search repo botkube` to see the charts.

## Contributing

The source code of the [BotKube](https://botkube.io) Helm charts can be found under the [`helm/botkube`](https://github.com/kubeshop/botkube/tree/main/helm/botkube) directory in the BotKube repository.

We'd love to have you contribute! Please refer to our [contribution guidelines](https://botkube.io/contribute/) for details.

## License

[MIT License](https://botkube.io/license/).
