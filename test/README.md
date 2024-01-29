# E2E Tests

This directory contains E2E tests. The tests instrument both Slack and Discord using a tester app and tester bot respectively.

Basically, our testers listen to events sent from Botkube in a test cluster. And, the testers also trigger commands for Botkube to execute.

On Kubernetes, the E2E tests are self-contained. They just require a Botkube installation on a cluster as highlighted in the instructions below.

## General prerequisites

- Kubernetes cluster (e.g. local one created with `k3d`)
- [Configure access to private `botkube-cloud` dependency](#configure-botkube-cloud-dependency-access) 

## Configure `botkube-cloud` dependency access

1. Export the `GOPRIVATE` environment variable:

   ```shell
   go env -w GOPRIVATE=github.com/kubeshop/botkube-cloud
   ```

2. Update your global git config file:

   ```text
   [url "git@github.com:kubeshop/botkube-cloud.git"]
    insteadOf = https://github.com/kubeshop/botkube-cloud
   ```

   Alternatively, if you prefer to use:

   ```text
   [url "git@github.com:"]
           insteadOf = https://github.com/
   ```

   > This new section informs Git that any URL starting with https://github.com/ should have that prefix replaced with ssh://git@github.com/ instead. Since Go uses HTTPS by default, this also affects your `go get` commands.

From now on, you can build and run end-to-end tests and update the version in `go.mod` by running:

```shell
go get -u github.com/kubeshop/botkube-cloud
```

If you are using IntelliJ, you can define the following environment variable for Go modules:

```shell
GOPRIVATE=github.com/kubeshop/botkube-cloud
```

## Testing Slack

### Prerequisites

- Botkube bot app configured for a Slack workspace according to the [instruction](https://docs.botkube.io/installation/slack/cloud-slack)
- Botkube tester app configured according to the [instruction](#configure-tester-slack-application)

### Configure Tester Slack application

> **NOTE:** This is something you need to do only once. Once the tester app is configured, you can use its token for running integration tests as many times as you want.

1. Create new Slack Application on [this page](https://api.slack.com/apps)
2. Use "From an app manifest" option
3. Copy and paste this manifest:

   ```yaml
   display_information:
     name: Botkube tester
   features:
     bot_user:
       display_name: Tester
       always_online: false
   oauth_config:
     scopes:
       bot:
         - channels:join
         - channels:manage
         - chat:write
         - users:read
         - channels:history
   settings:
     org_deploy_enabled: false
     socket_mode_enabled: false
     token_rotation_enabled: false
   ```

4. Install this app into your workspace
5. Navigate to the **OAuth & Permissions** section
6. Copy the **Bot User OAuth Token** and save it for later.

   You can already export it as environment variable for [Botkube installation](#install-botkube) or [running the tests](#run-the-tests):

   ```bash
   export SLACK_TESTER_APP_TOKEN="{Botkube tester app token}
   ```

## Testing Discord

### Prerequisites

- A Discord server available, [create one if required](https://support.discord.com/hc/en-us/articles/204849977-How-do-I-create-a-server-).
- Botkube bot app configured for a Discord server according to the [instruction](https://docs.botkube.io/installation/discord/self-hosted).
  > **NOTE:** Please name the app `botkube` and skip step 11 as it's not required.
- Botkube tester bot app configured according to the [instruction](#configure-tester-discord-bot-application).

### Configure Tester Discord bot application

> **NOTE:** This is something you need to do only once. Once the tester app is configured, you can use its token for running integration tests as many times as you want.

1. Create a new Discord app [here](https://discordapp.com/developers/applications).
1. Name the app: `botkube_tester`.
1. Navigate to the Bot page.

   1. Toggle `PUBLIC BOT` to OFF.
   1. Toggle `MESSAGE CONTENT INTENT` to ON.
   1. Click the "Reset Token" button.
   1. Copy the Token and export it as the `DISCORD_TESTER_APP_TOKEN` environment variable.

      ```bash
      export DISCORD_TESTER_APP_TOKEN="{Botkube Discord tester app bot token}"
      ```

1. Click on the "OAuth2" menu section, and navigate to the "URL Generator" page.
   1. Select `SCOPES` as `bot`.
   1. Select `BOT PERMISSIONS` as:
      - Manage Channels.
      - Read Messages/View Channels.
      - Send Messages.
      - Manage Messages.
      - Embed Links.
      - Attach Files.
      - Read Message History.
      - Mention Everyone.
   1. Generate the URL using the OAuth2 URL Generator available under the OAuth2 section to add bot to your Discord server.
   1. Copy the generated URL.
1. Paste the generated URL in a new tab, select the discord server to which you want to add the bot, click Continue and Authorize Bot addition.
1. Find the name of the server in the top left.
1. Right click and select `CopyID` to copy the server ID. This is the `DISCORD_GUILD_ID` that we'll need to run tests against the server.

   ```bash
   export DISCORD_GUILD_ID="{Botkube Discord tester guildID}"
   ```

## Install Botkube

Use environment vars for the specific platform (Slack or Discord or both) when running your E2E tests.

For example, if you're only running Discord tests, you can omit env var prefixed with `SLACK_..`.

1. Export required environment variables:

   ```bash
   #
   # Environment variables for running Botkube
   #

   export SLACK_BOT_TOKEN="{token for your configured Slack Botkube app}" # WARNING: Token for Botkube Slack bot, not the Tester!

   export DISCORD_BOT_ID="{Botkube Discord bot ClientID}" # WARNING: ClientID for Botkube Discord bot, not the Tester bot!
   export DISCORD_BOT_TOKEN="{token for your configured Discord Botkube bot}" # WARNING: Token for Botkube Discord bot, not the Tester!

   export IMAGE_REGISTRY="ghcr.io"
   export IMAGE_REPOSITORY="kubeshop/botkube"
   export IMAGE_TAG="v9.99.9-dev"

   #
   # Environment variables for running integration tests
   #
   export SLACK_TESTER_APP_TOKEN="{Botkube Slack tester app token}" # WARNING: Token for Tester, not the Botkube Slack bot!
   export DISCORD_TESTER_APP_TOKEN="{Botkube Discord tester app token}" # WARNING: Token for Tester, not the Botkube Discord bot!
   export DISCORD_GUILD_ID="{Discord server ID}" # Where the tests will

   #
   # Optional environment variables
   #
   export SLACK_TESTER_NAME="{Name of Botkube SLACK tester app}" # WARNING: tester name defaults to `tester` when a name is not provided for local test runs!
   export DISCORD_TESTER_NAME="{Name of Botkube DISCORD tester app}" # WARNING: tester name defaults to `tester` when a name is not provided for local test runs!
   ```

2. Install Botkube using Helm chart:

   ```bash
   helm install botkube --namespace botkube ./helm/botkube --wait --create-namespace \
     -f ./helm/botkube/e2e-test-values.yaml \
     --set communications.default-group.slack.token="${SLACK_BOT_TOKEN}" \
     --set communications.default-group.discord.token="${DISCORD_BOT_TOKEN}" \
     --set communications.default-group.discord.botID="${DISCORD_BOT_ID}" \
     --set image.registry="${IMAGE_REGISTRY}" \
     --set image.repository="${IMAGE_REPOSITORY}" \
     --set image.tag="${IMAGE_TAG}"
   ```

## Compile plugins

We test also the Botkube plugins system. To compile Botkube plugins, run:

```bash
OUTPUT_MODE="archive" make build-plugins
```

By default, built plugins' binaries are available under `plugin-dist` directory. The e2e framework builds the plugins index file dynamically and starts the HTTP server that is later accessible from the k3d cluster.

To override default settings, export following environment variables:

```bash
export PLUGINS_BINARIES_DIRECTORY="./plugin-dist"
export PLUGINS_SERVER_HOST="http://host.k3d.internal" # on K3d enabling you to access your host system by referring to it as host.k3d.internal
export PLUGINS_SERVER_PORT="3000"
```

## Run the tests

1. Ensure the environment variables for your target platforms are exported:

   ```bash
   export KUBECONFIG=/Users/$USER/.kube/config # set custom path if necessary
   export SLACK_TESTER_APP_TOKEN="{Botkube Slack tester app token}" # WARNING: Token for Tester, not the Botkube Slack bot!
   export DISCORD_TESTER_APP_TOKEN="{Botkube Discord tester app token}" # WARNING: Token for Tester, not the Botkube Discord bot!
   export DISCORD_GUILD_ID="{Discord server ID}"
   ```

2. Run the tests for Slack:
   ```bash
   make test-integration-slack
   ```
3. Run the tests for Discord:

   ```bash
   make test-integration-discord
   ```

### Clear `botkube-system` ConfigMap between test runs

Botkube tracks whether the initial help message was sent or not to minimise spam. This is tracked in a `botkube-system` ConfigMap.

After running the tests for an e2e target, e.g. `make test-integration-slack`, please ensure this ConfigMap is removed before rerunning the test against another target, e.g. `make test-integration-discord`.

```bash
kubectl delete cm botkube-system -n botkube # or the namespace where Botkube is installed
```

If you don't remove the ConfigMap, any e2e tests looking to verify that a help message is displayed will error. This also stops the rest of the e2e tests from running.

## Testing Cloud Slack end-to-end with Botkube Cloud

You can test Botkube Cloud Slack with Botkube Cloud. Follow the instructions below to set up the test environment and run the tests.

### Setting up test environment

1. Create Slack user for testing purposes with access to a given Slack workspace.
1. Create Botkube Cloud user with two organizations: FREE and TEAM one (with Cloud Slack quota).
1. Follow the [Testing Slack](#testing-slack) instructions to set up "Tester" bot.

### Running tests

To run the tests, get all the noted data from previous steps and export them as environment variables.

```bash
# Required
export SLACK_WORKSPACE_NAME="" # e.g. my-workspace
# The test log ins to the Slack workspace using the credentials below
export SLACK_EMAIL="" # e.g. my-email@example.com
export SLACK_PASSWORD=""
export SLACK_TESTER_BOT_NAME="" # e.g. botkubedev
export SLACK_TESTER_TESTER_BOT_TOKEN="" # e.g. xoxb-...
export BOTKUBE_CLOUD_EMAIL="" # e.g. my-email@example.com
export BOTKUBE_CLOUD_PASSWORD=""
export BOTKUBE_CLOUD_TEAM_ORGANIZATION_ID="" # e.g. 204a2d86-265c-4ae2-89a8-928f823ec4da
export BOTKUBE_CLOUD_FREE_ORGANIZATION_ID="" # e.g. c03bd605-7b8d-490f-b4d5-57c8a0560e83

# Optional - useful for running tests locally
export KUBECONFIG="" # path to your kubeconfig
export BOTKUBE_CLOUD_API_BASE_URL="" e.g. http://localhost:8080
export BOTKUBE_CLOUD_API_SLACK_APP_INSTALLATION_BASE_URL_OVERRIDE="" # provide if necessary e.g. using ngrok: https://d5ac-194-33-77-250.ngrok-free.app
export SLACK_BOT_DISPLAY_NAME="" # e.g. BotkubeDev
export SLACK_WORKSPACE_ALREADY_CONNECTED="false"
export SLACK_DISCONNECT_WORKSPACE_AFTER_TESTS="true"
export PAGE_TIMEOUT="1m"
export SCREENSHOTS_ENABLED="false" # disable screenshots
```

To run the test in headless mode (without browser window), run:

```shell
make test-cloud-slack-dev-e2e
```

To run the tests with Chromium browser window visible, run:

```shell
test-cloud-slack-dev-e2e-show-browser
```

Refer to the `E2ESlackConfig` (`./cloud-slack-dev-e2e/cloud_slack_dev_e2e_test.go`) for all possible environment variables.
