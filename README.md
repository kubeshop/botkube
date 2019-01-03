# botkube
[![Build Status](https://travis-ci.org/infracloudio/botkube.svg?branch=master)](https://travis-ci.org/infracloudio/botkube)

A slack bot which watches your kubernetes clusters and notifies about resources life cycles, errors, events and provide recommendations about the best practices while creating resources.
You can also ask botkube to execute kubectl commands on k8s cluster which helps debugging a application or cluster.

## Building
```
build/docker.sh <docker-repo> <tag>
```

## Creating slack bot
- Create a bot with name `botkube` https://my.slack.com/services/new/bot
- Set icon, name and copy API token (required to configure botkube)
- Add the bot into a channel by inviting with `@botkube` in the message area

## Configuration
Kubeops reads configurations from `config.yaml` file placed at `CONFIG_PATH`
e.g https://github.com/infracloudio/botkube/config.yaml

Supported resources for configuration:
- pods
- nodes
- services
- namespaces
- replicationcontrollers
- persistentvolumes
- persistentvolumeclaims
- secrets
- configmaps
- deployments
- daemonsets
- replicasets
- ingresses
- jobs
- roles
- rolebindings
- clusterroles
- clusterrolebindings

## Installing on kubernetes cluster
### Using helm
- Follow https://docs.helm.sh/using_helm/#installing-helm guide to install helm.
- Clone the botkube github repository.
```
git clone https://github.com/infracloudio/botkube.git
```
- Update default `botkubeconfig` in `helm/botkube/values.yaml` to watch the resources you want.
- Deploy botkube using `helm install` in your cluster.
```
helm install --name botkube --namespace botkube --set botkubeconfig.communications.slack.channel={SLACK_CHANNEL_NAME} --set botkubeconfig.communications.slack.token={SLACK_API_TOKEN_FOR_THE_BOT} helm/botkube/
```
- Send `@botkube help` in the channel to see if `botkube` is responding.
