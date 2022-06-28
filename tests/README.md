# E2E Tests

This directory contains E2E tests which are run against BotKube installed on Kubernetes cluster and Slack API.

## Prerequisites

- Kubernetes cluster (e.g. local one created with `k3d`)
- BotKube bot app configured for a given Slack workspace according to the [instruction](https://www.botkube.io/installation/slack/)
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

  You can already export it as environment variable for [running the tests locally](#run-tests-locally):

  ```bash
  export SLACK_TESTER_APP_TOKEN="{BotKube tester app token}
  ```

## Run tests locally

To run the tests manually against the latest development version, follow these steps:

1. Install BotKube using Helm chart:
        
    ```bash
    export SLACK_BOT_TOKEN="{token for your configured BotKube app}" # WARNING: It is token for BotKube Slack bot, not the Tester!
    cat > /tmp/values.yaml << ENDOFFILE
    communications:
      slack:
        enabled: false # Tests will override this temporarily
        token: ${SLACK_BOT_TOKEN} # Provide a valid token for BotKube app
        channel: botkube-test # Tests will override this temporarily
    config:
      resources:
        - name: v1/configmaps
          namespaces:
            include:
              - botkube
          events:
            - create
            - update
            - delete
        - name: v1/pods
          namespaces:
            include:
              - botkube
          events:
            - create
      settings: 
        clustername: sample
        kubectl:
          enabled: true
        upgradeNotifier: false
      enabled: true
    extraAnnotations:
      botkube.io/disable: "true"
    image:
      tag: v9.99.9-dev
    ENDOFFILE
    
    helm install botkube --namespace botkube ./helm/botkube -f /tmp/values.yaml --wait
    ```


1. Export required environment variables:

    ```bash
    export SLACK_TESTER_APP_TOKEN="{BotKube tester app token}" # WARNING: This is a token for Tester, not the BotKube Slack bot.
    export KUBECONFIG=/Users/$USER/.kube/config # set custom path if necessary
    ```

1. Run the tests and wait for the result:

    ```bash
    make test-integration
    ```
