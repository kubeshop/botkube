package bot

import (
	"fmt"
	"regexp"

	"github.com/kubeshop/botkube/pkg/config"
)

const slackBotMentionPrefixFmt = "^<@%s>"

func slackChannelsConfigFrom(channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) map[string]channelConfigByName {
	channels := make(map[string]channelConfigByName)
	for _, channCfg := range channelsCfg {
		channels[channCfg.Identifier()] = channelConfigByName{
			ChannelBindingsByName: channCfg,
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
