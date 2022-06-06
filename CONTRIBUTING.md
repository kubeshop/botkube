# How to Contribute to BotKube

We'd love your help!

BotKube is [MIT Licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contributions accepted.

We gratefully welcome improvements to [documentation](https://www.botkube.io/ "Go to documentation site") as well as to code.

## Contributing to documentation

You can contribute to documentation by following [these instructions](https://github.com/infracloudio/botkube-docs#contributing "Contributing to BotKube Docs")

## Compile BotKube from source code

Before you proceed, make sure you have installed BotKube Slack/Mattermost/Teams app and copied the required token as per the steps documented [here](https://www.botkube.io/installation/)

### Prerequisite

* Make sure you have [`go 1.18`](https://go.dev) installed.

* You will also need `make` and [`docker`](https://docs.docker.com/install/) installed on your
machine.
* Clone the source code
   ```sh
   $ git clone https://github.com/infracloudio/botkube.git
   ```

Now you can build and run BotKube by one of the following ways

### Build the container image

1. This will build BotKube and create a new container image tagged as `ghcr.io/infracloudio/botkube:v9.99.9-dev`
   ```sh
   $ make build
   $ make container-image
   $ docker tag ghcr.io/infracloudio/botkube:v9.99.9-dev-amd64 <your_account>/botkube:v9.99.9-dev
   $ docker push <your_account>/botkube:v9.99.9-dev
   ```
   Where `<your_account>` is Docker hub account to which you can push the image

2. Deploy newly created image in your cluster.

   a. Using helm (v3)

   ```sh
   $ helm repo add infracloudio https://infracloudio.github.io/charts
   $ helm repo update
   $ kubectl create namespace botkube
   $ helm install --version v9.99.9-dev botkube --namespace botkube \
     --set communications.slack.enabled=true \
     --set communications.slack.channel=<SLACK_CHANNEL_NAME> \
     --set communications.slack.token=<SLACK_API_TOKEN_FOR_THE_BOT> \
     --set settings.clustername=<CLUSTER_NAME> \
     --set settings.kubectl.enabled=<ALLOW_KUBECTL> \
     --set image.repository=<your_account>/botkube \
     --set image.tag=v9.99.9-dev \
     infracloudio/botkube
   ```

   Check [values.yaml](https://github.com/infracloudio/botkube/blob/develop/helm/botkube/values.yaml) for default options

   > Note:
   >
   > If you are using helm version < 3.0.0, use following command
   >
   > helm install --version v9.99.9-dev --name botkube --namespace botkube --set \<options\> infracloudio/botkube

   b. Using kubectl

     1. Edit deploy-all-in-one.yaml and update the configuration.
        Set SLACK_ENABLED, SLACK_CHANNEL, SLACK_API_TOKEN, clustername, kubectl.enabled and update the resource events configuration you want to receive notifications for in the configmap.
     2. Create botkube namespace and deploy resources
     ```sh
     $ kubectl create ns botkube && kubectl create -f deploy-all-in-one.yaml -n botkube
     ```

### Build and run BotKube locally

For faster development, you can also build and run BotKube outside K8s cluster.

1. Build BotKube binary if you don't want to build the container image, you can build the binary like this,
   ```sh
   # Fetch the dependencies
   $ go mod download
   # Build the binary
   $ go build ./cmd/botkube/
   ```
2. Edit `./resource_config.yaml` and `./comm_config.yaml` to configure resource and set communication credentials.

3. Export the path to directory of `config.yaml`
   ```sh
   # From project root directory
   $ export CONFIG_PATH=$(pwd)
   ```
4. Export the path to Kubeconfig:
    
   ```sh
   export KUBECONFIG=/Users/$USER/.kube/config # set custom path if necessary
   ```

5. Make sure that correct context is set and you are able to access your Kubernetes cluster
   ```console
   $ kubectl config current-context
   minikube
   $ kubectl cluster-info
   Kubernetes master is running at https://192.168.39.233:8443
   CoreDNS is running at https://192.168.39.233:8443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
   ...
   ```
6. Run BotKube binary
   ```sh
   $ ./botkube
   ```

## Making A Change

* Before making any significant changes, please [open an issue](https://github.com/infracloudio/botkube/issues). Discussing your proposed changes ahead of time will make the contribution process smooth for everyone.

* Once we've discussed your changes and you've got your code ready, make sure that the build steps mentioned above pass. Open your pull request against [`develop`](http://github.com/infracloudio/botkube/tree/develop) branch.

* To avoid build failures in CI, install [`golangci-lint` v1.46](https://golangci-lint.run/usage/install/) and run:
  ```sh
  # From project root directory
  $ make lint
  ```
  This will run the `golangci-lint` tool to lint the Go code.

* Run e2e tests
  ```sh
  $ ./hack/runtests.sh
  ```

* Make sure your pull request has [good commit messages](https://chris.beams.io/posts/git-commit/):
  * Separate subject from body with a blank line
  * Limit the subject line to 50 characters
  * Capitalize the subject line
  * Do not end the subject line with a period
  * Use the imperative mood in the subject line
  * Wrap the body at 72 characters
  * Use the body to explain _what_ and _why_ instead of _how_

* Try to squash unimportant commits and rebase your changes on to develop branch, this will make sure we have clean log of changes.
