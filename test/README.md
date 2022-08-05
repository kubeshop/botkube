# E2E Tests

This directory contains E2E tests which are run against BotKube installed on Kubernetes cluster and Slack API.

## Prerequisites

- Kubernetes cluster (e.g. local one created with `k3d`)
- BotKube bot app configured for a given Slack workspace according to the [instruction](https://botkube.io/installation/slack/)
- BotKube tester app configured according to the [instruction](#configure-tester-slack-application)

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

## Install BotKube

1. Export required environment variables:

    ```bash
    export SLACK_BOT_TOKEN="{token for your configured BotKube app}" # WARNING: It is token for BotKube Slack bot, not the Tester!
    export IMAGE_REGISTRY="ghcr.io"
    export IMAGE_REPOSITORY="kubeshop/botkube"
    export IMAGE_TAG="v9.99.9-dev"

    #
    # The following environmental variables are required only when running integration tests via Helm:
    #
    export SLACK_TESTER_APP_TOKEN="{BotKube tester app token}" # WARNING: This is a token for Tester, not the BotKube Slack bot!
    export TEST_IMAGE_REGISTRY="ghcr.io"
    export TEST_IMAGE_REPOSITORY="kubeshop/botkube-test"
    export TEST_IMAGE_TAG="v9.99.9-dev"
    ```

1. Install BotKube using Helm chart:

    ```bash
    helm install botkube --namespace botkube ./helm/botkube --wait --create-namespace \
      -f ./helm/botkube/e2e-test-values.yaml \
      --set communications.default-group.slack.token="${SLACK_BOT_TOKEN}" \
      --set image.registry="${IMAGE_REGISTRY}" \
      --set image.repository="${IMAGE_REPOSITORY}" \
      --set image.tag="${IMAGE_TAG}" \
      --set e2eTest.image.registry="${TEST_IMAGE_REGISTRY}" \
      --set e2eTest.image.repository="${TEST_IMAGE_REPOSITORY}" \
      --set e2eTest.image.tag="${TEST_IMAGE_TAG}" \
      --set e2eTest.slack.testerAppToken="${SLACK_TESTER_APP_TOKEN}"
    ```

## Run tests locally

1. Export required environment variables:

    ```bash
    export SLACK_TESTER_APP_TOKEN="{BotKube tester app token}" # WARNING: This is a token for Tester, not the BotKube Slack bot.
    export KUBECONFIG=/Users/$USER/.kube/config # set custom path if necessary
    ```

1. Run the tests and wait for the result:

    ```bash
    make test-integration
    ```

## Run Helm test

Follow these steps to run integration tests via Helm:

1. Run the tests:

    ```bash
    helm test botkube --namespace botkube --timeout=10m  --logs
    ```

1. Wait for the results. The logs will be printed when the `helm test` command exits.

    If you would like to see the logs in the real time, run:

    ```bash
    kubectl logs -n botkube botkube-e2e-test -f
    ```
