# botkube
[![Build Status](https://travis-ci.org/infracloudio/botkube.svg?branch=master)](https://travis-ci.org/infracloudio/botkube) [![Go Report Card](https://goreportcard.com/badge/github.com/infracloudio/botkube)](https://goreportcard.com/report/github.com/infracloudio/botkube)

A slack bot which keeps eye on your kubernetes resources and notifies about resources life cycles events, errors and warnings. It also provides you recommendations about the best practices while creating a resources.
You can also ask botkube to execute kubectl commands on k8s cluster which helps debugging a application or cluster.

## Getting started
### Install botkube app to your slack workspace
Click the "Add to Slack" button provided to install `botkube` slack application to your workspace. Once you authorized the application, you will be provided a BOT Access token. Kindly note down that token which will be required while deploying botkube controller to your cluster

<a href="https://slack.com/oauth/authorize?scope=bot&client_id=12637824912.515475697794"><img alt="Add to Slack" height="40" width="139" src="https://platform.slack-edge.com/img/add_to_slack.png" srcset="https://platform.slack-edge.com/img/add_to_slack.png 1x, https://platform.slack-edge.com/img/add_to_slack@2x.png 2x" /></a>

### Add botkube to a slack channel
After installing botkube app to your slack workspace, you could see new bot user with name 'botkube' create in your workspace. Add that bot to a slack channel you want to receive notification in. (You can add it by inviting using `@botkube` message in a required channel)

### Installing botkube controller to your kubernetes cluster
#### Using helm
- We will be using `helm` to install our k8s controller. Follow https://docs.helm.sh/using_helm/#installing-helm guide to install helm if you don't have it installed already
- Clone the botkube github repository.
```
git clone https://github.com/infracloudio/botkube.git
```
- Update default `config` in `helm/botkube/values.yaml` to watch the resources you want. (by default you will receive `create`, `delete` and `error` events for all the resources in all the namespaces
- Deploy botkube using `helm install` in your cluster.
```
helm install --name botkube --namespace botkube --set config.communications.slack.channel={SLACK_CHANNEL_NAME} --set config.communications.slack.token={SLACK_API_TOKEN_FOR_THE_BOT} helm/botkube/
```
- Send `@botkube help` in the channel to see if `botkube` is responding.

#### Configuration
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

## Notification screenshots
### notifications about resource creation

### notifications about resource deletion

## Executing kubectl commands using @botkube
@botkube also allows you to executes kubectl commands on k8s cluster and get output. Since botkube is created with service token having READONLY clusterrole, you can execute only READONLY kubectl commands.
Send `@botkube help` in slack chennel or directly to `botkube` user to find more information about the supported commands.

#### print help

#### get pods -n a namespace

#### get logs of a pod

## Building botkube controller
Use following command to build docker image for botkube controller
```
build/docker.sh <docker-repo> <tag>
```
