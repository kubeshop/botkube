package bot

import (
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

const slackBotMentionPrefixFmt = "^<@%s>"

func slackChannelsConfigFrom(log logrus.FieldLogger, channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) map[string]channelConfigByName {
	channels := make(map[string]channelConfigByName)
	for channAlias, channCfg := range channelsCfg {
		normalizedChannelName, changed := normalizeChannelName(channCfg.Name)
		if changed {
			log.Warnf("Channel name %q has been normalized to %q", channCfg.Name, normalizedChannelName)
		}
		channCfg.Name = normalizedChannelName

		channels[channCfg.Identifier()] = channelConfigByName{
			ChannelBindingsByName: channCfg,
			alias:                 channAlias,
			notify:                !channCfg.Notification.Disabled,
		}
	}

	return channels
}

func slackBotMentionRegex(botID string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(slackBotMentionPrefixFmt, botID))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}
