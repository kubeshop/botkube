# Event Manager Extensibility

Created on `2022-09-28` by Huseyin BABAL ([@huseyinbabal](https://github.com/huseyinbabal)))

| Status                                   |
|------------------------------------------|
| `PROPOSED/REJECTED/ACCEPTED/IMPLEMENTED` |

## Overview

<!--
General overview of the proposal and section with ToC
-->

<!-- toc -->
- [Motivation](#motivation)
  * [Goal](#goal)
  * [Non-goal](#non-goal)
- [Proposal](#proposal)
  * [Terminology](#terminology)
  * [Golang and Extensibility](#golang-and-extensibility)
  * [Hashicorp's Go Plugin](#hashicorps-go-plugin)
  * [Source Plugin Structure](#source-plugin-structure)
  * [Executor Plugin Structure](#executor-plugin-structure)
  * [Cloud Events](#cloud-events)
    + [BotKube Cloudevents Data Structure](#botkube-cloudevents-data-structure)

<!-- tocstop -->
## Motivation
Currently, we have support for 5 integrations for BotKube and BotKube is designed to listen events from kubernetes
and execute kubectl commands. In this design documentation, we aim to provide an architecture where end users can 
extend BotKube to have their integrations with respective configurations. Even this doc is initiated to extend event 
manager, having a general extension/plugin mechanism will bring a huge value to BotKube

### Goal
1. Introduce a feature for sources so that BotKube can accept events from sources other than Kubernetes.
2. Introduce a feature for executors so that BotKube can handle custom commands via extensions.

### Non-goal
1. Initially, we don't need to cover executor flow physically, but with a proper architecture, we can cover both events (coming from sources) executors

## Proposal
### Terminology
There are already a bunch of design patterns that help you to extend your system with additional capabilities. For example,
with proper dependency injection system, you can define an interface for the contract and implement different algorithms
based on that. In BotKube we might have a `Source` interface, and `KubernetesSource`|`PrometheusSource` would
be good candidates for concrete implementations. with the help of Dependency Injection, you can decide on which implementation
to call in runtime.

![](./assets/dependency-injection.png)

Based on the above diagram, BotKube only knows `Source` interface, and calls `Consume()` method to execute the logic in 
the real implementation. For example, if you register `KubernetesSource` implementation, BotKube can consume kubernetes
events. In same way, `PrometheusSource` can consume prometheus events, or it can add an endpoint to BotKube to let Prometheus
send alerts to this endpoint. Those examples can be extended, but is there another way to add those extensions in more detached way?
For example, instead of forcing to implement interfaces, what about we add plugins, and they are being called automatically by
main process (BotKube in our case). By doing that, plugins and core BotKube implementation can be independent and easy to maintain.

### Golang and Extensibility
Golang has [plugin](https://pkg.go.dev/plugin) package which you can use to add an extension system to your existing Go project.
This package has some limitations as follows;
- Once a problem occurs in plugin, it also crashes the host process (your main program, which is BotKube in our case).
- You can load the plugin during the initialization, then reload is not supported.
- You need to maintain conflict of shared libraries.

### Hashicorp's Go Plugin
There are several alternatives to plugin systems in Golang, but I want to focus specifically on 
[Hashicorp's Go Plugin](https://github.com/hashicorp/go-plugin) system. In this plugin system, main application talks to 
plugin via RPC. BotKube is a client in this case, and it calls plugin implementation via RPC and uses response returned from 
plugin. It basically uses file descriptor in local file system to communicate like a local network connection. 

![](./assets/hashicorp-go-plugin-architecture.png)

### Source Plugin Structure
In above diagram, RPC is the base protocol and in our case it is possible to use gRPC. If we think `KubernetesSource` again, 
BotKube can connect to `KubernetesSource` plugin and think about this is a server-side streaming. The kubernetes consuming logic is
located in that plugin and whenever it gets an event from kubernetes, it can stream it to client which is BotKube. The main
advantage of this usage, we can use any language to implement a plugin which supports gRPC. 

![](./assets/kubernetes-source-plugin.png)

Source plugin does not necessarily need to implement a logic since some sources may want to have just configuration fot the plugin in
BotKube. Instead of source logic, they may want to send events directly to BotKube
with a payload in [cloud events schema format](https://cloudevents.io/) which we will see soon. 

Events with cloudevents format can be easily consumed by BotKube once we have the handler. However, we may need to have custom
handler like Prometheus. Prometheus has a special event format, and we can redirect that events to specific destination with 
Prometheus job configuration. In Source Plugin architecture, it can also spin up a custom handler instead to handle prometheus 
event payload and process it in BotKube. This is a typical example to show how we can have a dynamic handling mechanism for Source Plugins.

To sum up source plugin system;
- They can consume external system like kubernetes events
- They can be in noop format, and they would have only plugin configuration to register that in BotKube
- They can also spin up an handler to handle custom events from other sources like Prometheus


### Executor Plugin Structure
Executor plugins would have nearly same flow as Source Plugins with a small difference. Executor plugins would be triggered
whenever BotKube receives a command from end-user. So, once BotKube receives command, based on the command prefix it can 
resolve which plugin to call and uses response on gRPC execution. Think about `KubectlExecutor`, once BotKube gets `kubectl get pods`
it can resolve plugin by its prefix which is `kubectl` and call `KubectlExecutor` gRPC endpoint automatically and parses the response.
As you can also see, BotKube does not need to know how `kubectl` works, it simply delegates all the responsibility to specific plugin.

![](./assets/kubectl-executor-plugin.png)

### Cloud Events
Cloudevents schema has a proven standard for the cloud native industry where well-known products uses this schema to distribute their events
via this contract. The main take of this, you don't need to re-invent wheel from scratch while you are designing your data format. In BotKube,
we can also use [one of the Cloudevents SDK](https://github.com/cloudevents/sdk-go) to spin up an handler inside BotKube so that any cloudevents 
compatible source can send data to BotKube. The main example to this is [Keptn](https://keptn.sh/), an open source application life-cycle orchestration tool, 
they send events in cloudevents format. If we want to ship Keptn events to BotKube to be able to send notifications, we may need to add new integration to Keptn
so that it will send events to BotKube `/events` endpoint which handles cloudevents.

#### BotKube Cloudevents Data Structure
You will see very simple suggested event format for BotKube as shown below.
```json
{
  "id" : "e2361318-e50b-472e-8b17-13dbac1daac1",   
  "source" : "kubernetes",     
  "specversion" : "1.0",                             
  "time" : "2022-03-23T14:35:39.738Z",               
  "data" : {                                         
    "namespace" : "botkube",
    "deployment" : {
      "name" : "test",
      "replicas" : 5
    }
  }
}
```

`id`: This is the unique identifier for the event, and this should be decided by end users. UUID is not a mandatory stuff, it can
be also ARN for example `ec2:myinstance:12345`.

`source`: This refers to plugin name. Once BotKube receives and event on `/events` endpoint, `source` field would be validated
to see we already have a registered plugin for this event or not. 

`specversion`: To have proper versioning for BotKube event format.

`time`: Event time

`data`: This is the event data that contains important information about source. what you see in `data` section is just example,
we can also add some kind of validation in BotKube to better use received data. For example, if we force user to send key value pairs,
we can easily apply templating/filtering to this event to full-fill our future stories about sending custom notifications to slack.
Data structure is strongly opened to suggestion.

## PoC
### Motivation
The main motivation of the PoC is to come up with a simple playground project that explains how it looks like to have a plugin system for BotKube.

### Folder Structure
![](./assets/plugin-system.png)
The plugin system contains 2 packages: `contrib` and `plugin`. `contrib` package is for end users and they can come up with their own
plugins to include them after PR approval. `plugin` package contains the logic to manage BotKube plugins. Let's deep dive those folders
to understand them a bit better.

### Contrib Folder
#### `build`: This contains plugin executables. Hashicorp's Go Plugin uses an RPC mechanism between client (BotKube), and server (Plugin),
and server side is actually an executable. Those executables can be maintained in a separate repo, and they can be cloned to local during BotKube 
startup process. Let's keep this in mind as an alternative plugin management notation.

#### `executors`: This package contains executor plugins. They are just Hashicorp Go Plugins that contains a main method that serves plugin implementation.
Plugin implementation is completely detached from BotKube system. Executor plugin has a contract `Execute(command string) (string, error)` where, BotKube
application can send command as an input, and executor plugin can return error or successful response from execution.

#### `sources`: Same as executors, and it has `Consume(ch chan interface) error` as a contract to be able to feed provided channel whenever there is an event 
reached to source plugin system.

#### `Makefiles`: They contain the logic of building plugins.

### Plugin Folder
#### `executor/proto`: Basically, we have **executor** and **source** plugins and they have their interfaces which are `Consume` and `Execute`. Proto folder contains 
the proto message definitions for executor plugins. In the root Makefile, you can see we generate Go code out of those messages. These means, whoever interested, they can
contribute to generate also for other languages and they can implement a plugin in other languages!

#### `source/proto`: Same as executor, it has proto message definitions for Consume operation. The main difference between executor and source is, source plugin 
is a server-side streaming plugin where it streams events to client which is BotKube in our case.

#### `**/grpc.go`: This contains the gRPC client/server definitions which Hashicorp Go Plugin system will use for RPC communication between BotKube and Plugin.
#### `**/interface.go`: Interface of plugin, it contains interface methods to describe plugin.
#### `**/plugin.go`: Contains plugin definition for Hashicorp Go Plugin system. Plugin definition contains the actual implementation behind that specific plugin.
#### `plugin/manager.go`: This is responsible for plugin management. The scenario for manager is as follows;
- It checks all the executables under `contrib` folder and aggregates them as list of plugin. 
- It creates a plugin client for each plugin by using either one of the Source or Executor plugin format.
- It registers all the plugin on initialization, and BotKube can resolve any of them on event, or command coming from platform and calls contract function which are 
`Consume` or `Execute`.

### Example
You can see a working example [here](https://github.com/huseyinbabal/botkube-plugins-playground)
<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
