# Mattermost Support

#### Assumptions
- Mattermost is already installed.
- Mattermost server IP is reachable from the cluster.

### Summary
`Mattermost` is an open source, self-hosted Slack-alternative and we want to add support for the same in Botkube.

### Motivation
Currently Botkube is supporting Slack, and as a feature addition we want to include `Mattermost` support. Botkube will run on the clusters and send notifications and alerts to the configured Mattermost team. It will also be able to execute `@botkube` commands from `Mattermost`. 

### Design
Steps for adding Mattermost support:
- Add package for Mattermost referring to `mattermost-server` repository.
- Add Mattermost configurations in helm chart or config.yaml.
- While starting controller, check in config if Mattermost support is enabled.
- If support enabled, initialize Mattermost with values from config.
- Start a goroutine for Mattermost.
- In controller, add notifier for `SendEvent` and `SendMessage` for Mattermost.

#### Adding package
The [mattermost-server](https://github.com/mattermost/mattermost-server) repository contains a client package to perform operations on `Mattermost` server. We will perform below operations using the package.
- Connecting to Mattermost server.
- Create `Botkube` user in Mattermost.
- Finding team in Mattermost and adding `Botkube` user to the Team.
- Finding channel or create channel if needed and adding `Botkube` user to the Channel.
- Send and Receive messages from Mattermost server.

#### Adding Configurations
We need to add below values for Mattermost in Helm Chart or config.yaml.
- Mattermost Support enabled/disabled flag
- Mattermost Server IP
- Team Name
- User-Email
- User-Name
- Password
- Channel Name(Same as Slack)

#### Controller Modifications
- In `main.go`, check in config if Mattermost support is enabled.
- If yes, then get config and add a goroutine for Mattermost.
- In `controller.go`, for event notifications and sending start/stop messages, add notifier for Mattermost.

### References
- https://github.com/mattermost/mattermost-bot-sample-golang
- https://github.com/mattermost/mattermost-server/blob/master/model/client4.go
- https://docs.mattermost.com/install/install-ubuntu-1804.html
