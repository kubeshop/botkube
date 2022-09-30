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
  * [User Story Name](#user-story-name)
    + [Suggested solution](#suggested-solution)
    + [Alternatives](#alternatives)
- [Consequences](#consequences)

<!-- tocstop -->
## Motivation
Currently we have support for 5 integrations for BotKube and BotKube is designed to listen events from kubernetes
and execute kubectl commands. In this design documentation, we aim to provide an architecture where end users can 
extend BotKube to have their integrations with respective configurations.

### Goal
1. Introduce a system for sources so that BotKube can accept events from sources other than Kubernetes.
2. Introduce a system for executors so that BotKube can handle custom commands via extensions.

### Non-goal
1. 
## Proposal

### User Story Name

#### Suggested solution

#### Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Consequences
