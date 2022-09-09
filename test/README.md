# E2E Tests

This directory contains E2E tests. The tests instrument both Slack and Discord using a tester app and tester bot respectively.

Basically, our testers listen to events sent from BotKube in a test cluster. And, the testers also trigger commands for BotKube to execute.

On Kubernetes, the E2E tests are self-contained. They just require a BotKube installation on a cluster as highlighted in the instructions below. 

## Prerequisites

- Kubernetes cluster (e.g. local one created with `k3d`)

### Slack

- BotKube bot app configured for a Slack workspace according to the [instruction](https://botkube.io/docs/installation/slack/)
- BotKube tester app configured according to the [instruction](#configure-tester-slack-application)

### Discord

- A Discord server available, [create one if required](https://support.discord.com/hc/en-us/articles/204849977-How-do-I-create-a-server-).
- BotKube bot app configured for a Discord server according to the [instruction](https://botkube.io/docs/installation/discord/#install-botkube-to-the-discord-server)
  > **NOTE:** Please name the app `botkube` and skip step 11 as it's not required.
- BotKube tester bot app configured according to the [instruction](#configure-tester-discord-bot-application)

### Configure Tester Slack application

> **NOTE:** This is something you need to do only once. Once the tester app is configured, you can use its token for running integration tests as many times as you want.

1. Create new Slack Application on [this page](https://api.slack.com/apps)
2. Use "From an app manifest" option
3. Copy and paste this manifest:

    ```yaml
    display_information:
      name: BotKube tester
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

   You can already export it as environment variable for [BotKube installation](#install-botkube) or [running the tests locally](#run-tests-locally):

   ```bash
   export SLACK_TESTER_APP_TOKEN="{BotKube tester app token}
   ```

### Configure Tester Discord bot application

> **NOTE:** This is something you need to do only once. Once the tester app is configured, you can use its token for running integration tests as many times as you want.

1. Create a new Discord app [here](https://discordapp.com/developers/applications).
2. Name the app: `botkube_tester`.
3. Navigate to the Bot page and Click Add Bot to add a Discord Bot to your application.
4. Click the Reset Token button.
5. Copy the Token and export it as the `DISCORD_TESTER_APP_TOKEN` environment variable.
   ```bash
   export DISCORD_TESTER_APP_TOKEN="{BotKube Discord tester app bot token}
   ```
6. Go to the OAuth2 page.
7. Select `SCOPES` as `bot`.
8. Select `BOT PERMISSIONS` as:
   - Manage Channels.
   - Read Messages/View Channels.
   - Send Messages.
   - Manage Messages.
   - Embed Links.
   - Attach Files.
   - Read Message History.
   - Mention Everyone.
9. Generate the URL using the OAuth2 URL Generator available under the OAuth2 section to add bot to your Discord server.
10. Copy and Paste the generated URL in a new tab, select the discord server to which you want to add the bot, click Continue and Authorize Bot addition.
11. Go back to the Discord screen and navigate to the Bot page.
12. Toggle `PUBLIC BOT` to OFF and `MESSAGE CONTENT INTENT` to ON.
13. Navigate back to the Discord server where you've installed tester app bot.
14. Find the name of the server in the top left.
15. Right click and select `CopyID` to copy the server ID. This is the `DISCORD_GUILD_ID` that we'll need to run tests against the server.
   ```bash
   export DISCORD_GUILD_ID="{BotKube Discord tester guildID}
   ```

## Install BotKube

1. Export required environment variables:

    ```bash
    export SLACK_BOT_TOKEN="{token for your configured Slack BotKube app}" # WARNING: Token for BotKube Slack bot, not the Tester!
    
    export DISCORD_BOT_ID="{BotKube Discord bot ClientID}" # WARNING: ClientID for BotKube Discord bot, not the Tester bot!
    export DISCORD_BOT_TOKEN="{token for your configured Discord BotKube bot}" # WARNING: Token for BotKube Discord bot, not the Tester!

    export IMAGE_REGISTRY="ghcr.io"
    export IMAGE_REPOSITORY="kubeshop/botkube"
    export IMAGE_TAG="v9.99.9-dev"

    #
    # Environment variables for running integration tests via Helm:
    #
    export TEST_IMAGE_REGISTRY="ghcr.io"
    export TEST_IMAGE_REPOSITORY="kubeshop/botkube-test"
    export TEST_IMAGE_TAG="v9.99.9-dev"
    
    #
    # Environment variables for running integration tests both LOCALLY and via Helm:
    #
    export SLACK_TESTER_APP_TOKEN="{BotKube Slack tester app token}" # WARNING: Token for Tester, not the BotKube Slack bot!
    export DISCORD_TESTER_APP_TOKEN="{BotKube Discord tester app token}" # WARNING: Token for Tester, not the BotKube Discord bot!
    export DISCORD_GUILD_ID="{Discord server ID}" # Where the tests will
    
    #
    # Optional: environment variables for running integration tests LOCALLY using make:
    #
    export SLACK_TESTER_NAME="{Name of BotKube tester app}" # WARNING: tester name defaults to `tester` when a name is not provided for local test runs! 
    ```

2. Install BotKube using Helm chart:

    ```bash
    helm install botkube --namespace botkube ./helm/botkube --wait --create-namespace \
      -f ./helm/botkube/e2e-test-values.yaml \
      --set communications.default-group.slack.token="${SLACK_BOT_TOKEN}" \
      --set communications.default-group.discord.token="${DISCORD_BOT_TOKEN}" \
      --set communications.default-group.discord.botID="${DISCORD_BOT_ID}" \
      --set image.registry="${IMAGE_REGISTRY}" \
      --set image.repository="${IMAGE_REPOSITORY}" \
      --set image.tag="${IMAGE_TAG}" \
      --set e2eTest.image.registry="${TEST_IMAGE_REGISTRY}" \
      --set e2eTest.image.repository="${TEST_IMAGE_REPOSITORY}" \
      --set e2eTest.image.tag="${TEST_IMAGE_TAG}" \
      --set e2eTest.slack.testerAppToken="${SLACK_TESTER_APP_TOKEN}" \
      --set e2eTest.discord.testerAppToken="${DISCORD_TESTER_APP_TOKEN}" \
      --set e2eTest.discord.guildID="${DISCORD_GUILD_ID}"
    ```

## Run tests locally

1. Ensure these environment variables are exported:

    ```bash
    export SLACK_TESTER_APP_TOKEN="{BotKube Slack tester app token}" # WARNING: Token for Tester, not the BotKube Slack bot!
    export DISCORD_TESTER_APP_TOKEN="{BotKube Discord tester app token}" # WARNING: Token for Tester, not the BotKube Discord bot!
    export DISCORD_GUILD_ID="{Discord server ID}" # Where the tests will
    export KUBECONFIG=/Users/$USER/.kube/config # set custom path if necessary
    ```

2. Run the tests for Slack and Discord in parallel :

    ```bash
    make test-integration-slack & make test-integration-discord & 
    ```
 
## Run Helm test

Follow these steps to run integration tests via Helm:

1. Run the tests:

    ```bash
    helm test botkube --namespace botkube --timeout=30m  --logs
    ```

   Wait for the results. The logs will be printed when the `helm test` command exits.

2. If you would like to see the logs in real time, run:

    ```bash
    kubectl logs -n botkube botkube-e2e-test -f
    ```
