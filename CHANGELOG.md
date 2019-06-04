# Change Log

## [Unreleased](https://github.com/infracloudio/botkube/tree/HEAD)

[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.7.0...HEAD)

**Implemented enhancements:**

- Make develop as a default branch [\#91](https://github.com/infracloudio/botkube/issues/91)
- Image tag in Helm chart should be configurable [\#88](https://github.com/infracloudio/botkube/issues/88)
- Capability to disable specific filters [\#71](https://github.com/infracloudio/botkube/issues/71)
- Add GitHub Pull Request template [\#80](https://github.com/infracloudio/botkube/pull/80) ([bhavin192](https://github.com/bhavin192))
- Add CONTRIBUTING.md [\#79](https://github.com/infracloudio/botkube/pull/79) ([bhavin192](https://github.com/bhavin192))

**Fixed bugs:**

- \[BUG\] Fresh deploy in Kubernetes 1.14.1 not working [\#85](https://github.com/infracloudio/botkube/issues/85)
- \[BUG\] mattermost self-signed certificate [\#81](https://github.com/infracloudio/botkube/issues/81)

**Merged pull requests:**

- \[Travis CI\] don't push container image for PR builds [\#94](https://github.com/infracloudio/botkube/pull/94) ([bhavin192](https://github.com/bhavin192))
- Build and push docker image from develop [\#93](https://github.com/infracloudio/botkube/pull/93) ([PrasadG193](https://github.com/PrasadG193))
- Add support to manage filters with @BotKube command [\#84](https://github.com/infracloudio/botkube/pull/84) ([PrasadG193](https://github.com/PrasadG193))
- SSL support for Mattermost in Botkube [\#83](https://github.com/infracloudio/botkube/pull/83) ([arush-sal](https://github.com/arush-sal))

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
- Mattermost implementation changes for botkube channel. [\#76](https://github.com/infracloudio/botkube/pull/76) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Check cluster-name for ping command [\#75](https://github.com/infracloudio/botkube/pull/75) ([PrasadG193](https://github.com/PrasadG193))
- Add logic to restart BotKube pod if config file is updated [\#74](https://github.com/infracloudio/botkube/pull/74) ([PrasadG193](https://github.com/PrasadG193))
- Return BotKube version with response to ping [\#73](https://github.com/infracloudio/botkube/pull/73) ([PrasadG193](https://github.com/PrasadG193))
- Add new env var LOG\_LEVEL to set logging levels [\#68](https://github.com/infracloudio/botkube/pull/68) ([PrasadG193](https://github.com/PrasadG193))
- Add Skip var in Event struct to skip an event [\#67](https://github.com/infracloudio/botkube/pull/67) ([PrasadG193](https://github.com/PrasadG193))
- Hide sensitive info while displaying configuration [\#65](https://github.com/infracloudio/botkube/pull/65) ([PrasadG193](https://github.com/PrasadG193))
- Add badges for Slack and docs [\#61](https://github.com/infracloudio/botkube/pull/61) ([bhavin192](https://github.com/bhavin192))
- Add support for elasticsearch interface [\#59](https://github.com/infracloudio/botkube/pull/59) ([PrasadG193](https://github.com/PrasadG193))
- Add support for Mattermost [\#55](https://github.com/infracloudio/botkube/pull/55) ([mugdha-adhav](https://github.com/mugdha-adhav))
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

- Issue \#46: Removed @botkube help commands. Added Makefile to add support for git tags and docker build with versioning [\#45](https://github.com/infracloudio/botkube/pull/45) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Issue \#41: Updated Dockerfile for multi-stage build [\#44](https://github.com/infracloudio/botkube/pull/44) ([mugdha-adhav](https://github.com/mugdha-adhav))
- Create CODE\_OF\_CONDUCT.md [\#43](https://github.com/infracloudio/botkube/pull/43) ([ssudake21](https://github.com/ssudake21))
- Update issue templates [\#42](https://github.com/infracloudio/botkube/pull/42) ([ssudake21](https://github.com/ssudake21))

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
- Add arch diagram to README [\#29](https://github.com/infracloudio/botkube/pull/29) ([PrasadG193](https://github.com/PrasadG193))
- Serve BotKube commands without '@' mention in DMs  [\#25](https://github.com/infracloudio/botkube/pull/25) ([PrasadG193](https://github.com/PrasadG193))

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

- Change license [\#18](https://github.com/infracloudio/botkube/pull/18) ([PrasadG193](https://github.com/PrasadG193))
- Improve README [\#12](https://github.com/infracloudio/botkube/pull/12) ([PrasadG193](https://github.com/PrasadG193))

## [v0.2.0](https://github.com/infracloudio/botkube/tree/v0.2.0) (2019-01-15)
[Full Changelog](https://github.com/infracloudio/botkube/compare/v0.1.0...v0.2.0)

**Closed issues:**

- Add flag to enable/disable kubectl command execution [\#8](https://github.com/infracloudio/botkube/issues/8)

**Merged pull requests:**

- Add yaml specs to deploy with kubectl command [\#14](https://github.com/infracloudio/botkube/pull/14) ([PrasadG193](https://github.com/PrasadG193))
- Support notifications from multiple clusters [\#13](https://github.com/infracloudio/botkube/pull/13) ([PrasadG193](https://github.com/PrasadG193))

## [v0.1.0](https://github.com/infracloudio/botkube/tree/v0.1.0) (2019-01-11)
**Closed issues:**

- Change of name [\#4](https://github.com/infracloudio/botkube/issues/4)
- Publish messages when bot started/stopped [\#2](https://github.com/infracloudio/botkube/issues/2)

**Merged pull requests:**

- Add LICENSE [\#11](https://github.com/infracloudio/botkube/pull/11) ([PrasadG193](https://github.com/PrasadG193))
- Rename kubeops to botkube [\#5](https://github.com/infracloudio/botkube/pull/5) ([PrasadG193](https://github.com/PrasadG193))
- Send bot start/stop messages to slack channel [\#3](https://github.com/infracloudio/botkube/pull/3) ([PrasadG193](https://github.com/PrasadG193))
