# kubeops
[![Build Status](https://travis-ci.org/infracloudio/kubeops.svg?branch=master)](https://travis-ci.org/infracloudio/kubeops)

A slack bot which watches your kubernetes clusters and notifies about resources life cycles, errors, events and provide recommendations about the best practices while creating resources.
You can also ask kubeops to execute kubectl commands on k8s cluster which helps debugging a application or cluster.

## Building
```
build/docker.sh <docker-repo> <tag>
```

## Creating slack bot
- Create a bot with name `kubeops` https://my.slack.com/services/new/bot
- Set icon, name and copy API token (required to configure kubeops)
- Add the bot into a channel by inviting with `@kubeops` in the message area

## Configuration
Kubeops reads configurations from `kubeopsconfig.yaml` file placed at `KUBEOPS_CONFIG_PATH`
Syntax:
```
# Resources you want to watch
resources:
  - name: RESOURCE_NAME    # Name of the resources e.g pods, deployments, ingresses, etc. (Resource name must be in plural form)
    namespaces:            # List of namespaces, "all" will watch all the namespaces
      - all
    events:                # List lifecycle events you want to receive, e.g create, update, delete OR all
      - all
  - name: pods
    namespaces:
      - kube-system
      - default
    events:
      - create
      - delete
  - name: roles
    namespaces:
      - all
    events:
      - create
      - delete

# K8S errors/warnings events you want to receive for the configured resources
events:
  types:
    - normal
    - warning

# Check true if you want to receive recommendations
# about the best practices for the created resource
recommendations: true

# Channels configuration
communications:
  slack:
    channel: 'SLACK_CHANNEL_NAME'
    token: 'SLACK_API_TOKEN_FOR_THE_BOT'
```
Supported resources:
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
- Clone the kubeops github repository.
```
git clone https://github.com/infracloudio/kubeops.git
```
- Update default `kubeopsconfig` in `helm/kubeops/values.yaml` to watch the resources you want.
- Deploy kubeops using `helm install` in your cluster.
```
helm install --name kubeops --namespace kubeops --set kubeopsconfig.communications.slack.channel={SLACK_CHANNEL_NAME} --set kubeopsconfig.communications.slack.token={SLACK_API_TOKEN_FOR_THE_BOT} helm/kubeops/
```
- Send `@kubeops help` in the channel to see if `kubeops` is responding.
