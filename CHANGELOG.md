# Change Log

## [v0.9.1](https://github.com/infracloudio/botkube/tree/v0.9.1) (2019-11-25)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.9.0...v0.9.1)

**Fixed bugs:**

- \[BUG\] BotKube not notifying about cluster-scoped resource [\#218](https://github.com/infracloudio/botkube/issues/218)
- \[BUG\] BotKube stops responding to the commands after Slack update [\#216](https://github.com/infracloudio/botkube/issues/216)
- \[BUG\] botkube.io is not accessible  [\#210](https://github.com/infracloudio/botkube/issues/210)
- \[BUG\] ping message isn't filtered by Channel Id \(mattermost\) [\#205](https://github.com/infracloudio/botkube/issues/205)
- \[BUG\] Failed to list \*v1beta1.Ingress: the server could not find the requested resources [\#204](https://github.com/infracloudio/botkube/issues/204)
- \[Mattermost\] Bot stop responding to commands after a while [\#201](https://github.com/infracloudio/botkube/issues/201)
- \[BUG\] Botkube sends slack notifications multiple times for an event [\#181](https://github.com/infracloudio/botkube/issues/181)

**Merged pull requests:**

- Add colors to Mattermost notification attachments [\#223](https://github.com/infracloudio/botkube/pull/223) ([PrasadG193](https://github.com/PrasadG193))
- Add mattermost connection retry logic [\#222](https://github.com/infracloudio/botkube/pull/222) ([PrasadG193](https://github.com/PrasadG193))
- Fix cluster scoped resource notification issue [\#221](https://github.com/infracloudio/botkube/pull/221) ([PrasadG193](https://github.com/PrasadG193))
- Update slack dep to work with latest Slack api changes [\#217](https://github.com/infracloudio/botkube/pull/217) ([PrasadG193](https://github.com/PrasadG193))
- Remove errant } from clusterrole helm template [\#213](https://github.com/infracloudio/botkube/pull/213) ([baronomasia](https://github.com/baronomasia))
- Update image repo in release script [\#207](https://github.com/infracloudio/botkube/pull/207) ([PrasadG193](https://github.com/PrasadG193))
- Add pod security policy so botkube works in restricted clusters [\#195](https://github.com/infracloudio/botkube/pull/195) ([baronomasia](https://github.com/baronomasia))

## [v0.9.0](https://github.com/infracloudio/botkube/tree/v0.9.0) (2019-10-11)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.8.0...v0.9.0)

**Implemented enhancements:**

- Add basic proxy support for helm deployment [\#155](https://github.com/infracloudio/botkube/issues/155)
- Customizable \(or one-liner\) notifications [\#58](https://github.com/infracloudio/botkube/issues/58)
- Run as non-root [\#161](https://github.com/infracloudio/botkube/issues/161)
- \[Kubernetes/Helm\] Make cert usage generic [\#160](https://github.com/infracloudio/botkube/issues/160)
- Send Alert for Image Version Updates [\#151](https://github.com/infracloudio/botkube/issues/151)
- Ability to disable config file watcher [\#150](https://github.com/infracloudio/botkube/issues/150)
- Unit-Test code in Travis [\#144](https://github.com/infracloudio/botkube/issues/144)
- \[Refactoring\] Use SharedInformerFactory instead of cache.NewInformer to watch K8s resources [\#143](https://github.com/infracloudio/botkube/issues/143)
- Migrate BotKube to Go modules [\#137](https://github.com/infracloudio/botkube/issues/137)
- Improve test coverage for BotKube [\#136](https://github.com/infracloudio/botkube/issues/136)
- Show Docker Image Tag on Deployments [\#135](https://github.com/infracloudio/botkube/issues/135)
- Node Level Critical Events using filters [\#134](https://github.com/infracloudio/botkube/issues/134)
- Annotations based enable/disable notifications for a particular object. [\#133](https://github.com/infracloudio/botkube/issues/133)
- Annotations Based Multi-Channel support [\#132](https://github.com/infracloudio/botkube/issues/132)
- Send more info with update resource notification [\#131](https://github.com/infracloudio/botkube/issues/131)
- Exclude or Ignore Namespaces [\#128](https://github.com/infracloudio/botkube/issues/128)

**Fixed bugs:**

- \[BUG\] Remove Deprecated API groups in K8s 1.16 [\#191](https://github.com/infracloudio/botkube/issues/191)
- \[BUG\] File upload fails when output of "@BotKube log" it too long [\#185](https://github.com/infracloudio/botkube/issues/185)
- \[BUG\] empty namespaces in config file ignores all events [\#157](https://github.com/infracloudio/botkube/issues/157)
- \[BUG\] Update and Error events of old resources are skipped [\#147](https://github.com/infracloudio/botkube/issues/147)
- \[Openshift\] Pod keeps restarting as the registered watcher says "Config file /config/config.yaml is updated" [\#142](https://github.com/infracloudio/botkube/issues/142)
- \[BUG\] Botkube 0.8.0 - crash loop - invalid memory address or nil pointer deference [\#126](https://github.com/infracloudio/botkube/issues/126)
- Test cases missing [\#57](https://github.com/infracloudio/botkube/issues/57)

**Closed issues:**

- \[ERROR\] unmarshal error from configmap.yaml [\#194](https://github.com/infracloudio/botkube/issues/194)
- \[Cleanup\] Remove vendor [\#165](https://github.com/infracloudio/botkube/issues/165)

**Merged pull requests:**

- Fix namespace format in config files [\#198](https://github.com/infracloudio/botkube/pull/198) ([PrasadG193](https://github.com/PrasadG193))
- Add Latest Release Version Badge to README.md [\#196](https://github.com/infracloudio/botkube/pull/196) ([codenio](https://github.com/codenio))
- Update K8s package deps to 1.16 [\#193](https://github.com/infracloudio/botkube/pull/193) ([PrasadG193](https://github.com/PrasadG193))
- Update Deprecated API groups in K8s 1.16 [\#192](https://github.com/infracloudio/botkube/pull/192) ([PrasadG193](https://github.com/PrasadG193))
- Add sample config referenced in the docs [\#190](https://github.com/infracloudio/botkube/pull/190) ([PrasadG193](https://github.com/PrasadG193))
- Fix duplicate notification for Job update [\#187](https://github.com/infracloudio/botkube/pull/187) ([PrasadG193](https://github.com/PrasadG193))
- Fix file upload for Slack [\#186](https://github.com/infracloudio/botkube/pull/186) ([PrasadG193](https://github.com/PrasadG193))
- fix invalid memory address or nil pointer deference on mattermost [\#184](https://github.com/infracloudio/botkube/pull/184) ([gangseok514](https://github.com/gangseok514))
- \[CI\] Publish latest helm chart to helm chart repo [\#180](https://github.com/infracloudio/botkube/pull/180) ([PrasadG193](https://github.com/PrasadG193))
- Fix CI to build docker image [\#177](https://github.com/infracloudio/botkube/pull/177) ([PrasadG193](https://github.com/PrasadG193))
- Make notification messages more readable [\#175](https://github.com/infracloudio/botkube/pull/175) ([PrasadG193](https://github.com/PrasadG193))
- Refactor Test Suits [\#174](https://github.com/infracloudio/botkube/pull/174) ([codenio](https://github.com/codenio))
- Fix uninitialised filters and minor issues [\#173](https://github.com/infracloudio/botkube/pull/173) ([codenio](https://github.com/codenio))
- \[Documentation\] Add godoc reference badge in README [\#172](https://github.com/infracloudio/botkube/pull/172) ([PrasadG193](https://github.com/PrasadG193))
- Add support for Webhooks [\#169](https://github.com/infracloudio/botkube/pull/169) ([codenio](https://github.com/codenio))
- Run containers using Non Privileged user [\#168](https://github.com/infracloudio/botkube/pull/168) ([codenio](https://github.com/codenio))
- Remove Installation instructions from README [\#167](https://github.com/infracloudio/botkube/pull/167) ([sanketsudake](https://github.com/sanketsudake))
- \[cleanup\] Remove vendor [\#166](https://github.com/infracloudio/botkube/pull/166) ([PrasadG193](https://github.com/PrasadG193))
- Fix cluster field not being populated in ES [\#164](https://github.com/infracloudio/botkube/pull/164) ([codenio](https://github.com/codenio))
- \[Rebased\] Feature/generic ssl [\#163](https://github.com/infracloudio/botkube/pull/163) ([rajinator](https://github.com/rajinator))
- Node Level Critical Events filter [\#159](https://github.com/infracloudio/botkube/pull/159) ([codenio](https://github.com/codenio))
- Enhance Update Events with resource spec diff and Change event message formats [\#158](https://github.com/infracloudio/botkube/pull/158) ([codenio](https://github.com/codenio))
- add proxyURL, deployment env variable if-loop [\#156](https://github.com/infracloudio/botkube/pull/156) ([rajinator](https://github.com/rajinator))
- Add E2E Integration tests [\#154](https://github.com/infracloudio/botkube/pull/154) ([PrasadG193](https://github.com/PrasadG193))
- Unskip error and update events for old resources [\#148](https://github.com/infracloudio/botkube/pull/148) ([codenio](https://github.com/codenio))
- Use SharedInformerFactory instead of cache.Informer [\#146](https://github.com/infracloudio/botkube/pull/146) ([PrasadG193](https://github.com/PrasadG193))
- \[unit test\] Enable Unit test and add target in Makefile [\#145](https://github.com/infracloudio/botkube/pull/145) ([codenio](https://github.com/codenio))
- Fix minor bugs [\#140](https://github.com/infracloudio/botkube/pull/140) ([codenio](https://github.com/codenio))
- \[feature\] Add Object Annotation filter, Fixes \#132, \#133 [\#138](https://github.com/infracloudio/botkube/pull/138) ([codenio](https://github.com/codenio))
- Enhance Notification : short/long notification type [\#127](https://github.com/infracloudio/botkube/pull/127) ([codenio](https://github.com/codenio))

## [v0.8.0](https://github.com/infracloudio/botkube/tree/v0.8.0) (2019-07-09)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.7.0...v0.8.0)

**Implemented enhancements:**

- `BotKube filters list` should give one liner description about the listed filters [\#113](https://github.com/infracloudio/botkube/issues/113)
- Notify user to upgrade if newer version is available [\#110](https://github.com/infracloudio/botkube/issues/110)
- Add "error" option to resource level BotKube config [\#96](https://github.com/infracloudio/botkube/issues/96)
- Include job status \(succeeded/failed\) with notification [\#95](https://github.com/infracloudio/botkube/issues/95)
- Make develop as a default branch [\#91](https://github.com/infracloudio/botkube/issues/91)
- Make target for building BotKube binary [\#90](https://github.com/infracloudio/botkube/issues/90)
- Image tag in Helm chart should be configurable [\#88](https://github.com/infracloudio/botkube/issues/88)
- Capability to disable specific filters [\#71](https://github.com/infracloudio/botkube/issues/71)
- Using Github releases for BotKube [\#70](https://github.com/infracloudio/botkube/issues/70)
- Automatically generating changelog for BotKube [\#69](https://github.com/infracloudio/botkube/issues/69)
- Add GitHub Pull Request template [\#80](https://github.com/infracloudio/botkube/pull/80) ([bhavin192](https://github.com/bhavin192))
- Add CONTRIBUTING.md [\#79](https://github.com/infracloudio/botkube/pull/79) ([bhavin192](https://github.com/bhavin192))

**Fixed bugs:**

- \[BUG\] BotKube sends older update events after resync [\#123](https://github.com/infracloudio/botkube/issues/123)
- \[BUG\] BotKube should trim whitespaces in the command [\#121](https://github.com/infracloudio/botkube/issues/121)
- \[BUG\] Commands shouldn't need @BotKube prefix if posted as DM to BotKube user [\#115](https://github.com/infracloudio/botkube/issues/115)
- \[BUG\] Timestamp is missing in the BotKube error events [\#107](https://github.com/infracloudio/botkube/issues/107)
- \[BUG\] BotKube is sending older events when configured to watch update events [\#103](https://github.com/infracloudio/botkube/issues/103)
- \[BUG\] unable to find service account in clusterrolebinding. [\#102](https://github.com/infracloudio/botkube/issues/102)
- \[BUG\] crashloop for mattermost bot on new deploy [\#87](https://github.com/infracloudio/botkube/issues/87)
- \[BUG\] Fresh deploy in Kubernetes 1.14.1 not working [\#85](https://github.com/infracloudio/botkube/issues/85)
- \[BUG\] mattermost self-signed certificate [\#81](https://github.com/infracloudio/botkube/issues/81)

**Merged pull requests:**

- Add flag to control Config Watcher [\#152](https://github.com/infracloudio/botkube/pull/152) ([codenio](https://github.com/codenio))
- \[feature\] Add Support Go Modules, Remove dep dependencies [\#141](https://github.com/infracloudio/botkube/pull/141) ([codenio](https://github.com/codenio))
- \[feature\] Add support for ignoring namespaces [\#139](https://github.com/infracloudio/botkube/pull/139) ([codenio](https://github.com/codenio))
- Merge develop to master [\#125](https://github.com/infracloudio/botkube/pull/125) ([PrasadG193](https://github.com/PrasadG193))
- Fix timestamp for update events [\#124](https://github.com/infracloudio/botkube/pull/124) ([PrasadG193](https://github.com/PrasadG193))
- Trim whitespaces in the BotKube command [\#122](https://github.com/infracloudio/botkube/pull/122) ([PrasadG193](https://github.com/PrasadG193))
- Serve DMs to BotKube in Mattermost [\#120](https://github.com/infracloudio/botkube/pull/120) ([PrasadG193](https://github.com/PrasadG193))
- Create bot interface for communication mediums [\#119](https://github.com/infracloudio/botkube/pull/119) ([PrasadG193](https://github.com/PrasadG193))
- Patch/delete double whitespace [\#118](https://github.com/infracloudio/botkube/pull/118) ([nnao45](https://github.com/nnao45))
- Update CONTRIBUTING.md to add missing build step [\#117](https://github.com/infracloudio/botkube/pull/117) ([kingdevnl](https://github.com/kingdevnl))
- Treat DM as a valid incoming message [\#116](https://github.com/infracloudio/botkube/pull/116) ([PrasadG193](https://github.com/PrasadG193))
- Add description to the filters [\#114](https://github.com/infracloudio/botkube/pull/114) ([PrasadG193](https://github.com/PrasadG193))
- Fix make target in travis CI to build image [\#112](https://github.com/infracloudio/botkube/pull/112) ([PrasadG193](https://github.com/PrasadG193))
-  Check if newer version of BotKube is available and notify user  [\#111](https://github.com/infracloudio/botkube/pull/111) ([PrasadG193](https://github.com/PrasadG193))
- Use gothub for publishing releases [\#109](https://github.com/infracloudio/botkube/pull/109) ([PrasadG193](https://github.com/PrasadG193))
- Add JobStatusChecker filter and fix timestamp for error events [\#108](https://github.com/infracloudio/botkube/pull/108) ([PrasadG193](https://github.com/PrasadG193))
- Add script to publish release and auto generate CHANGELOG [\#101](https://github.com/infracloudio/botkube/pull/101) ([PrasadG193](https://github.com/PrasadG193))
- New make target `container-image` to build docker image [\#100](https://github.com/infracloudio/botkube/pull/100) ([vinayakshnd](https://github.com/vinayakshnd))
- Add CHANGELOG for older releases [\#99](https://github.com/infracloudio/botkube/pull/99) ([PrasadG193](https://github.com/PrasadG193))
- Add new event type "error" in config to watch error events on a resource [\#98](https://github.com/infracloudio/botkube/pull/98) ([PrasadG193](https://github.com/PrasadG193))
- \[Travis CI\] don't push container image for PR builds [\#94](https://github.com/infracloudio/botkube/pull/94) ([bhavin192](https://github.com/bhavin192))
- Build and push docker image from develop [\#93](https://github.com/infracloudio/botkube/pull/93) ([PrasadG193](https://github.com/PrasadG193))
- Add support to manage filters with @BotKube command [\#84](https://github.com/infracloudio/botkube/pull/84) ([PrasadG193](https://github.com/PrasadG193))
- SSL support for Mattermost in Botkube [\#83](https://github.com/infracloudio/botkube/pull/83) ([arush-sal](https://github.com/arush-sal))
- Add support for elasticsearch interface [\#59](https://github.com/infracloudio/botkube/pull/59) ([PrasadG193](https://github.com/PrasadG193))

## [v0.7.0](https://github.com/infracloudio/botkube/tree/v0.7.0) (2019-04-04)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.6.0...v0.7.0)

**Implemented enhancements:**

- Restart BotKube pod if configmap is updated [\#72](https://github.com/infracloudio/botkube/issues/72)
- Add Skip field BotKube Event  [\#63](https://github.com/infracloudio/botkube/issues/63)
- Add ElasticSearch support [\#53](https://github.com/infracloudio/botkube/issues/53)
- Mattermost support [\#26](https://github.com/infracloudio/botkube/issues/26)

**Fixed bugs:**

- \[BUG\] Add a way to set Log level [\#66](https://github.com/infracloudio/botkube/issues/66)
- \[BUG\] Do not dump auth token in `@BotKube notifier showconfig` command response [\#64](https://github.com/infracloudio/botkube/issues/64)
- Report Botkube and Kubernetes version in ping command [\#62](https://github.com/infracloudio/botkube/issues/62)
- \[BUG\] Helm chart version not in sync with Botkube version [\#51](https://github.com/infracloudio/botkube/issues/51)

**Closed issues:**

- Migrating to "Slash Commands" for `help` [\#46](https://github.com/infracloudio/botkube/issues/46)
- Add filter to add warnings if pod is created without any labels [\#27](https://github.com/infracloudio/botkube/issues/27)

**Merged pull requests:**

- \[helm chart\] make image tag configurable [\#89](https://github.com/infracloudio/botkube/pull/89) ([bhavin192](https://github.com/bhavin192))
- CHANGELOG.md [\#78](https://github.com/infracloudio/botkube/pull/78) ([PrasadG193](https://github.com/PrasadG193))
- merge develop to master [\#77](https://github.com/infracloudio/botkube/pull/77) ([PrasadG193](https://github.com/PrasadG193))
- Mattermost implementation changes for botkube channel. [\#76](https://github.com/infracloudio/botkube/pull/76) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Check cluster-name for ping command [\#75](https://github.com/infracloudio/botkube/pull/75) ([PrasadG193](https://github.com/PrasadG193))
- Add logic to restart BotKube pod if config file is updated [\#74](https://github.com/infracloudio/botkube/pull/74) ([PrasadG193](https://github.com/PrasadG193))
- Return BotKube version with response to ping [\#73](https://github.com/infracloudio/botkube/pull/73) ([PrasadG193](https://github.com/PrasadG193))
- Add new env var LOG\_LEVEL to set logging levels [\#68](https://github.com/infracloudio/botkube/pull/68) ([PrasadG193](https://github.com/PrasadG193))
- Add Skip var in Event struct to skip an event [\#67](https://github.com/infracloudio/botkube/pull/67) ([PrasadG193](https://github.com/PrasadG193))
- Hide sensitive info while displaying configuration [\#65](https://github.com/infracloudio/botkube/pull/65) ([PrasadG193](https://github.com/PrasadG193))
- Add badges for Slack and docs [\#61](https://github.com/infracloudio/botkube/pull/61) ([bhavin192](https://github.com/bhavin192))
- Issue \#51: Added target in Makefile to update helm version [\#54](https://github.com/infracloudio/botkube/pull/54) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Modified cluster flag to support quotes [\#52](https://github.com/infracloudio/botkube/pull/52) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Issue \#27: Added filter to add warning if pod created without label [\#48](https://github.com/infracloudio/botkube/pull/48) ([mugdha-adhav](https://github.com/mugdha-adhav))

## [v0.6.0](https://github.com/infracloudio/botkube/tree/v0.6.0) (2019-03-07)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.5.0...v0.6.0)

**Implemented enhancements:**

- Run the command on the slack channel, only the information of the corresponding cluster is displayed. [\#37](https://github.com/infracloudio/botkube/issues/37)
- Include clustername in start/stop messages from BotKube [\#16](https://github.com/infracloudio/botkube/issues/16)

**Fixed bugs:**

- Reduce docker image size [\#41](https://github.com/infracloudio/botkube/issues/41)

**Closed issues:**

- "BotKube notifier" commands should be executable only from a slack channel [\#40](https://github.com/infracloudio/botkube/issues/40)

**Merged pull requests:**

- Add support for Mattermost [\#55](https://github.com/infracloudio/botkube/pull/55) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Merge develop to master [\#49](https://github.com/infracloudio/botkube/pull/49) ([PrasadG193](https://github.com/PrasadG193))
- Merge master into develop [\#47](https://github.com/infracloudio/botkube/pull/47) ([PrasadG193](https://github.com/PrasadG193))
- Issue \#46: Removed @botkube help commands. Added Makefile to add support for git tags and docker build with versioning [\#45](https://github.com/infracloudio/botkube/pull/45) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Issue \#41: Updated Dockerfile for multi-stage build [\#44](https://github.com/infracloudio/botkube/pull/44) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Create CODE\_OF\_CONDUCT.md [\#43](https://github.com/infracloudio/botkube/pull/43) ([sanketsudake](https://github.com/sanketsudake))
- Update issue templates [\#42](https://github.com/infracloudio/botkube/pull/42) ([sanketsudake](https://github.com/sanketsudake))

## [v0.5.0](https://github.com/infracloudio/botkube/tree/v0.5.0) (2019-02-28)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.4.0...v0.5.0)

**Fixed bugs:**

- Add helm lint in travis CI [\#33](https://github.com/infracloudio/botkube/issues/33)

**Closed issues:**

- Add "helm lint" checks to CI build [\#34](https://github.com/infracloudio/botkube/issues/34)

**Merged pull requests:**

- Added check slack channel in config, when get message  [\#39](https://github.com/infracloudio/botkube/pull/39) ([gimmetm](https://github.com/gimmetm))
- Issues \#37 and \#16: Added multi-cluster support and added cluster-name in botkube commands [\#38](https://github.com/infracloudio/botkube/pull/38) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Add "helm lint" check to travis ci build [\#35](https://github.com/infracloudio/botkube/pull/35) ([PrasadG193](https://github.com/PrasadG193))
- Adding icon so chart passes `helm lint` [\#31](https://github.com/infracloudio/botkube/pull/31) ([adamhaney](https://github.com/adamhaney))
- Develop [\#30](https://github.com/infracloudio/botkube/pull/30) ([sanketsudake](https://github.com/sanketsudake))
- Add arch diagram to README [\#29](https://github.com/infracloudio/botkube/pull/29) ([PrasadG193](https://github.com/PrasadG193))
- Serve BotKube commands without '@' mention in DMs  [\#25](https://github.com/infracloudio/botkube/pull/25) ([PrasadG193](https://github.com/PrasadG193))
- Merge develop to master [\#22](https://github.com/infracloudio/botkube/pull/22) ([PrasadG193](https://github.com/PrasadG193))

## [v0.4.0](https://github.com/infracloudio/botkube/tree/v0.4.0) (2019-01-18)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.3.0...v0.4.0)

**Closed issues:**

- Change botkube to BotKube in help messages [\#20](https://github.com/infracloudio/botkube/issues/20)

**Merged pull requests:**

- Capitalize 's' in 'slack' in help messages [\#23](https://github.com/infracloudio/botkube/pull/23) ([PrasadG193](https://github.com/PrasadG193))
- Rename "botkube" to "BotKube" in help messages [\#21](https://github.com/infracloudio/botkube/pull/21) ([PrasadG193](https://github.com/PrasadG193))

## [v0.3.0](https://github.com/infracloudio/botkube/tree/v0.3.0) (2019-01-17)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.2.0...v0.3.0)

**Closed issues:**

- Change license to MIT [\#17](https://github.com/infracloudio/botkube/issues/17)
- Add license [\#10](https://github.com/infracloudio/botkube/issues/10)
- Provide a way to install without Helm [\#9](https://github.com/infracloudio/botkube/issues/9)
- Send clustername in the event message [\#7](https://github.com/infracloudio/botkube/issues/7)

**Merged pull requests:**

- Merge develop to master [\#19](https://github.com/infracloudio/botkube/pull/19) ([PrasadG193](https://github.com/PrasadG193))
- Change license [\#18](https://github.com/infracloudio/botkube/pull/18) ([PrasadG193](https://github.com/PrasadG193))
- Improve README [\#12](https://github.com/infracloudio/botkube/pull/12) ([PrasadG193](https://github.com/PrasadG193))

## [v0.2.0](https://github.com/infracloudio/botkube/tree/v0.2.0) (2019-01-15)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.1.0...v0.2.0)

**Closed issues:**

- Add flag to enable/disable kubectl command execution [\#8](https://github.com/infracloudio/botkube/issues/8)

**Merged pull requests:**

- Merge develop to master [\#15](https://github.com/infracloudio/botkube/pull/15) ([PrasadG193](https://github.com/PrasadG193))
- Add yaml specs to deploy with kubectl command [\#14](https://github.com/infracloudio/botkube/pull/14) ([PrasadG193](https://github.com/PrasadG193))
- Support notifications from multiple clusters [\#13](https://github.com/infracloudio/botkube/pull/13) ([PrasadG193](https://github.com/PrasadG193))

## [v0.1.0](https://github.com/infracloudio/botkube/tree/v0.1.0) (2019-01-11)
**Closed issues:**

- Change of name [\#4](https://github.com/infracloudio/botkube/issues/4)
- Publish messages when bot started/stopped [\#2](https://github.com/infracloudio/botkube/issues/2)

**Merged pull requests:**

- Add LICENSE [\#11](https://github.com/infracloudio/botkube/pull/11) ([PrasadG193](https://github.com/PrasadG193))
- merge develop to master [\#6](https://github.com/infracloudio/botkube/pull/6) ([PrasadG193](https://github.com/PrasadG193))
- Rename kubeops to botkube [\#5](https://github.com/infracloudio/botkube/pull/5) ([PrasadG193](https://github.com/PrasadG193))
- Send bot start/stop messages to slack channel [\#3](https://github.com/infracloudio/botkube/pull/3) ([PrasadG193](https://github.com/PrasadG193))
- Develop [\#1](https://github.com/infracloudio/botkube/pull/1) ([sanketsudake](https://github.com/sanketsudake))



