# How to Contribute to BotKube

We'd love your help!

BotKube is [MIT Licensed](LICENSE) and accepts contributions via
GitHub pull requests. This document outlines some of the conventions
on development workflow, commit message formatting, contact points and
other resources to make it easier to get your contributions accepted.

We gratefully welcome improvements to
[documentation](https://www.botkube.io/ "Go to documentation site") as
well as to code.

## Contributing to documentation
You can contribute to documentation by following [these
instructions](https://github.com/infracloudio/botkube-docs#contributing
"Contributing to BotKube Docs")

## Compile BotKube from source code
### Prerequisite
* Make sure you have `go` compiler installed.
* BotKube uses [`dep`](https://github.com/golang/dep) to manage
dependencies.
* You will also need `make` and
[`docker`](https://docs.docker.com/install/) installed on your
machine.
* Clone the source code
   ```sh
   $ git clone https://github.com/infracloudio/botkube.git $GOPATH/src/github.com/infracloudio/botkube
   $ cd $GOPATH/src/github.com/infracloudio/botkube
   ```

Now you can build and run BotKube by one of the following ways
### Build the container image
1. This will build BotKube and create a new container image tagged as `infracloud/botkube:latest`
   ```sh
   $ make build
   $ docker tag infracloud/botkube:latest <your_account>/botkube:latest
   $ docker push <your_account>/botkube:latest
   ```
   Where `<your_account>` is Docker hub account to which you can push the image

2. Follow the [instructions from
   README.md](https://github.com/infracloudio/botkube#using-helm) to
   deploy newly created image in your cluster.
   ```sh
   # Set the container image
   $ helm install --name botkube --namespace botkube \
	   ... other options ...
	   --set image.repository=your_account/botkube \
	   --set image.tag=latest
	   ...
	   helm/botkube
   ```

### Build and run BotKube locally
1. Build BotKube binary  
   If you don't want to build the container image, you can build the
   binary like this,
   ```sh
   # Fetch the dependencies
   $ dep ensure
   # Build the binary
   $ go build ./cmd/botkube/
   ```
2. Modify `config.yaml` according to your needs. Please refer
   [configuration section](https://www.botkube.io/configuration/) from
   documentation for more details
   ```sh
   # From project root directory
   $ xdg-open config.yaml
   ```
3. Export the path to directory of `config.yaml`
   ```sh
   # From project root directory
   $ export CONFIG_PATH=$(pwd)
   ```
4. Make sure that correct context is set and you are able to access
   your Kubernetes cluster
   ```console
   $ kubectl config current-context
   minikube
   $ kubectl cluster-info
   Kubernetes master is running at https://192.168.39.233:8443
   CoreDNS is running at https://192.168.39.233:8443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
   ...
	```
5. Run BotKube binary
   ```sh
   $ ./botkube
   ```

## Making A Change

* *Before making any significant changes, please [open an
issue](https://github.com/infracloudio/botkube/issues).* Discussing
your proposed changes ahead of time will make the contribution process
smooth for everyone.

* Once we've discussed your changes and you've got your code ready,
make sure that build steps mentioned above pass. Open your pull
request against
[`develop`](http://github.com/infracloudio/botkube/tree/develop)
branch.

* To avoid build failures in CI, run
  ```sh
  # From project root directory
  $ ./hack/verify-*.sh
  ```
  This will check if the code is properly formatted, linted & vendor directory is present.

* Make sure your pull request has [good commit
messages](https://chris.beams.io/posts/git-commit/):
  * Separate subject from body with a blank line
  * Limit the subject line to 50 characters
  * Capitalize the subject line
  * Do not end the subject line with a period
  * Use the imperative mood in the subject line
  * Wrap the body at 72 characters
  * Use the body to explain _what_ and _why_ instead of _how_

* Try to squash unimportant commits and rebase your changes on to
develop branch, this will make sure we have clean log of changes.
