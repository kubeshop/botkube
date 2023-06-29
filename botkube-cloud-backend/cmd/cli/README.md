# Migration tool for Botkube Cloud

Command line tool that helps you migrate your Botkube installation to Botkube Cloud.

## Installation

```bash
go build -o bctl main.go
```

## Usage

We assume you have a working Botkube instance and a Botkube Cloud account.
This tool gathers all the information needed to migrate your Botkube instance from your Kubernetes
cluster - a working kube config is also needed.

1. Find the namespace where the Botkube instance is installed (`botkube` is the default):

```bash
kubectl get ns
kubectl get pod -n botkube --show-labels
```

2. Login to Botkube Cloud:

```bash
bctl login
```

3. Run the migration tool and follow instructions:

```bash
bctl migrate --namespace botkube --labels app=botkube
```

## Implementation details

### Login

We tried to make the migration process as simple and automated as possible.
The login workflow involves a locally served http server that listens for a callback from the browser
after the user login. The callback contains the access token that is used to authenticate the user
and is stored locally in `~/.botkube/cloud.json`.
The server is stopped after the callback is received.

### Migration

Once logged in, we create a pod in the same namespace as the Botkube instance that mounts the same
secrets and config maps as the Botkube pod and generates and stores the entire configuration in a
config map `botkube-migration`.

Once we have the configuration, we can turn it into a API call and create identical
resources in Botkube Cloud.
