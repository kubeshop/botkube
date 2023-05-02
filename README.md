<p align="center">
  <a href="https://botkube.io">
    <img src="./docs/assets/botkube-title.png" alt="Botkube Logo Light" />
  </a>
</p>

<p align="center">
  Botkube is a messaging bot for monitoring and debugging Kubernetes clusters.
</p>


<p align="center">
  <a href="https://github.com/kubeshop/botkube/releases/latest">
    <img src="https://img.shields.io/github/v/release/kubeshop/botkube" alt="Latest Release" />
  </a>
  <a href="https://github.com/kubeshop/botkube/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/kubeshop/botkube" alt="License"/>
  </a>
  <a href="https://join.botkube.io/">
    <img src="https://badgen.net/badge/slack/Botkube?icon=slack" alt="Slack" />
  </a>
  <a href="https://github.com/kubeshop/botkube/actions?query=workflow%3ACI+branch%3Amain">
    <img src="https://github.com/kubeshop/botkube/workflows/CI/badge.svg?branch=main" alt="CI Build" />
  </a>
  <a href="https://godoc.org/github.com/kubeshop/botkube">
    <img src="https://godoc.org/github.com/kubeshop/botkube?status.svg" alt="Go Docs" />
  </a>
</p>

## Overview

Botkube helps you monitor your Kubernetes cluster, debug critical deployments and gives recommendations for standard practices by running checks on the Kubernetes resources. It integrates with multiple communication platforms, such as [Slack](https://slack.com), [Discord](https://discord.com/), or [Mattermost](https://mattermost.com).

You can also execute `kubectl` commands on K8s cluster via Botkube which helps debugging an application or cluster.

<p align="center">
<img src="./docs/assets/main-demo.gif" />
</p>

## Getting started

Please follow [this](https://docs.botkube.io/installation/) for a complete Botkube installation guide.

## Documentation

For full documentation, visit [botkube.io](https://docs.botkube.io). The documentation sources reside on the [botkube-docs](https://github.com/kubeshop/botkube-docs) repository under **content** directory.

## Features

<img src="./docs/assets/icons/terminal-box-line.svg" width="12%" align="right"/>

### Execute `kubectl` commands

The same `kubectl` capabilities inside your favorite communicator. You do not have to learn anything new! Plus, you can configure which `kubectl` commands Botkube can execute. See [configuration](https://docs.botkube.io/configuration/) for details.

<br /><br />

<img src="./docs/assets/icons/question-answer-line.svg" width="10%" align="left"/>

### Use multiple communication platforms

Botkube integrates with Slack, Discord, Mattermost, Microsoft Teams, ElasticSearch and outgoing webhook. See [configuration](https://docs.botkube.io/configuration/communication/) syntax for details.

<br /><br />

<img src="./docs/assets/icons/stack-line.svg" width="13%" align="right"/>

### Monitor any Kubernetes resource

Botkube supports literally any Kubernetes resource, including Custom Resources. For example, if you use [`cert-manager`](https://cert-manager.io/), you can get alerted about certificate issue, or backup failure in case you use backup tools like [Velero](https://velero.io/) or [Kanister](https://kanister.io/).

<br /><br />

<img src="./docs/assets/icons/bug-line.svg" width="12%" align="left"/>

### Debug anywhere, anytime

Using Botkube you can debug your apps deployed on Kubernetes from anywhere. To extract crucial information from the cluster, you can even use mobile communicator apps, like Slack. The entire team can see what steps have already been taken and avoid duplicated work.

<br /><br />

<img src="./docs/assets/icons/cloud-line.svg" width="12%" align="right"/>

### Deploy on any Kubernetes cluster

You can deploy Botkube backend on any Kubernetes cluster. It doesn't matter whether it is [K3d](https://k3d.io), managed Kubernetes on a cloud provider, or bare-metal one.

## Licence

This project is currently licensed under the [MIT License](https://github.com/kubeshop/botkube/blob/main/LICENSE).
