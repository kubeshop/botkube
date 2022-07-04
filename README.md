# BotKube

[![CI](https://github.com/kubeshop/botkube/workflows/CI/badge.svg?branch=main)](https://github.com/kubeshop/botkube/actions?query=workflow%3ACI+branch%3Amain) [![Go Report Card](https://goreportcard.com/badge/github.com/kubeshop/botkube)](https://goreportcard.com/report/github.com/kubeshop/botkube)
[![BotKube website](https://img.shields.io/badge/docs-botkube.io-blue.svg)](https://botkube.io)
[![GoDoc](https://godoc.org/github.com/kubeshop/botkube?status.svg)](https://godoc.org/github.com/kubeshop/botkube)
[![Release Version](https://img.shields.io/github/v/release/kubeshop/botkube?label=Botkube)](https://github.com/kubeshop/botkube/releases/latest)
[![License](https://img.shields.io/github/license/kubeshop/botkube?color=light%20green&logo=github)](https://github.com/kubeshop/botkube/blob/main/LICENSE)
[![Slack](https://badgen.net/badge/slack/BotKube?icon=slack)](http://join.botkube.io/)


For complete documentation visit www.botkube.io

BotKube integration with [Slack](https://slack.com), [Mattermost](https://mattermost.com) or [Microsoft Teams](https://www.microsoft.com/microsoft-365/microsoft-teams/group-chat-software) helps you monitor your Kubernetes cluster, debug critical deployments and gives recommendations for standard practices by running checks on the Kubernetes resources.
You can also ask BotKube to execute kubectl commands on k8s cluster which helps debugging an application or cluster.

![](botkube-title.jpg)

## Getting started
Please follow [this](https://www.botkube.io/installation/) for a complete BotKube installation guide.

## Architecture
![](/botkube_arch.jpg)
- **Informer Controller:** Registers informers to kube-apiserver to watch events on the configured k8s resources. It forwards the incoming k8s event to the Event Manager.
- **Event Manager:** Extracts required fields from k8s event object and creates a new BotKube event struct. It passes BotKube event struct to the Filter Engine.
- **Filter Engine:** Takes the k8s object and BotKube event struct and runs Filters on them. Each filter runs some validations on the k8s object and modifies the messages in the BotKube event struct if required.
- **Event Notifier:** Finally, notifier sends BotKube event over the configured communication channel.
- **Bot Interface:** Bot interface takes care of authenticating and managing connections with communication mediums like Slack, Mattermost, Microsoft Teams and reads/sends messages from/to them.
- **Executor:** Executes BotKube or kubectl command and sends back the result to the Bot interface.

Visit www.botkube.io for Configuration, Usage and Examples.

## Licence

This project is currently licensed under the [MIT License](https://github.com/kubeshop/botkube/blob/main/LICENSE).
