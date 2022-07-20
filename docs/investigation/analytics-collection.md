# Analytics collection

Test

This document contains a summary of a short research around collecting anonymous analytics (a.k.a. telemetry) for BotKube.

## Goal

The main goal is learning how BotKube is used, in the most private and least invasive way for users.
All data must be anonymized to prevent identifying users. All analytics will be used only as aggregated collection of data (in forms of some statistics), to further improve BotKube and adjust its roadmap.

The change must be widely communicated to all users, and there must be an ability to opt out from collecting such analytics.

## How others do it

I tried to look for some software from CNCF landscape which has telemetry built-in. Unfortunately, I couldn't find many examples when it comes to this field. It was much easier to find something in JavaScript world (e.g. NextJS).

Anyway, here are a few examples I found:

- [`odo`](https://github.com/redhat-developer/odo) - Twilio Segment ([source](https://github.com/redhat-developer/odo/blob/77d6b6df5cdd05074db8728d1aead76d3e259e25/pkg/segment/segment.go)),
- [`config-syncer` (previously `kubed`)](https://github.com/kubeops/config-syncer) - Google Analytics ([source](https://github.com/kubeops/config-syncer/blob/release-0.12/vendor/kmodules.xyz/client-go/tools/cli/cli.go)),
- [`netdata`](https://github.com/netdata/netdata) - PostHog ([see docs](https://learn.netdata.cloud/docs/agent/anonymous-statistics)),
- [Testkube](https://github.com/kubeshop/testkube) - Google Analytics 4 ([source](https://github.com/kubeshop/testkube/blob/34c57dbdb9312b68910e0ad5808485292fa31313/pkg/analytics/analytics.go)),
- Rancher backed off from this idea when it comes to k3s: [see issue](https://github.com/k3s-io/k3s/issues/834).

All of the software I found that collect anonymous analytics (not only from Go/Kubernetes space) have opt-out based configuration.

## Data collection

This list contains initial data which we will collect.

- Running installations:
  - BotKube Version,
  - Kubernetes version (it might be helpful to know which K8s we should support),
  - Enabled notifiers (names),
  - Enabled bots (names).
- Executed commands:
  - For `kubectl` commands:
    - exclude namespace (it could potentially identify user),
    - exclude resource name (it could potentially identify user),
    - collect just command verbs.
- Received events:
  - See how many events are handled by BotKube and sent to any communication channel,
    - We should batch the events to avoid high resource consumption during heavy load. This could be a built-in feature, like in [Segment Go library](https://segment.com/docs/connections/sources/catalog/libraries/server/go/#batching).
- Errors:
  - Application crashes (from inside the app),
  - Configuration errors (syntax, missing fields etc.),
  - Notification errors (when e.g. there was an error to post Slack or Discord message).

The data collection would be enabled by default.
By using two different identifiers at the same time, we will aggregate the data:
- per cluster (by using `uid` of the `kube-system` Namespace),
- and per BotKube installation (generating and storing UUID during BotKube installation e.g. in a ConfigMap).

### Out of scope

- Enabled filters (as they will be probably reworked in future). Based on run commands we will be still able to tell which filters are toggled on/off.

## Technical solution

After doing a little research, we could use the following solutions:

- **Application:** Twilio Segment

  - a single API for analytics events, which can be forwarded to multiple destinations,
  - multiple integrations with external APIs,
  - ability to correlate events from different sources (e.g. website and appplication),
  - automatic detection of the PII data,
  - paid solution.

  While Google Analytics 4 could be also used for sending the events from the app (for free), Segment has some advantages. For example, see the [discussion](https://www.reddit.com/r/GoogleTagManager/comments/f8nicy/how_is_segment_and_other_tools_different_to_gtm/) on Reddit.

- **Website:** Google Analytics 4

  - completely free solution,
  - powerful,
  - we can use the [Google Analytics integration](https://segment.com/catalog/integrations/google-analytics/) to poll the data into Segment. Alternatively, we could use Segment integration directly on the website, but it would be more expensive, and I think it is not worth it for now.

### Alternatives

- Piwik Pro - more privacy-friendly alternative to Google Analytics; however, the free plan is not as generous as GA. No Go API, but it is not a problem as it exposes HTTP API.
- Self-hosted solutions (e.g. [PostHog](https://github.com/PostHog/posthog)) - too big effort vs little benefit to set it up.
- (web only) [Cloudflare Web analytics](https://www.cloudflare.com/web-analytics/) - for websites set up behind Cloudflare; cookie banner not required; requires a paid plan.

## Consequences

Once agreed, the following to do list will be pasted into the [epic issue (#506)](https://github.com/kubeshop/botkube/issues/506):

- [ ] Set up Google Analytics and Twilio Segment accounts
  - Ensure proper access rights

- [ ] Implement collecting analytics in BotKube
  - Collect analytics listed in [Data collection](#data-collection) section
  - Use [Segment Go Library](https://segment.com/docs/connections/sources/catalog/libraries/server/go/)
  - Enabled by default
  - Make sure users can opt out:
    - a dedicated env variable supported by app
    - a dedicated Helm chart property
  - Document changes on botkube.io:
    - Update [privacy policy](https://github.com/kubeshop/botkube-docs/blob/main/content/privacy.md) and describe data we collect
    - Describe configuration - how to opt-out
      - Consider describing it also for each installation document

- [ ] Implement analytics on the website
  - Use Google Analytics 4
  - If needed: Add cookie banner (some reports say Google Analytics 4 are cookieless, but I also read on official website that `gtag.js` library uses first-party cookies, so a user consent might be also needed)
  - Document changes: Update [privacy policy](https://github.com/kubeshop/botkube-docs/blob/main/content/privacy.md) and describe data we collect

- [ ] Communicate changes
  - Write announcement on Slack to gather feedback before/during implementation (before 0.13 release)
    - Explain the reason of introducing the analytics
    - Point to the [epic issue](https://github.com/kubeshop/botkube/issues/506) for further discussion
    - If necessary, schedule a public meeting to discuss the changes
  - Mention analytics in BotKube 0.13 GitHub release description
