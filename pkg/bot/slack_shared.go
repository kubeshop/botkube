package bot

import (
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/config"
	conversationx "github.com/kubeshop/botkube/pkg/conversation"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

const slackBotMentionPrefixFmt = "^<@%s>"

func slackChannelsConfigFrom(log logrus.FieldLogger, channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) map[string]channelConfigByName {
	channels := make(map[string]channelConfigByName)
	for channAlias, channCfg := range channelsCfg {
		normalizedChannelName, changed := conversationx.NormalizeChannelIdentifier(channCfg.Name)
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

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	Text            string
	Channel         string
	ThreadTimeStamp string
	UserID          string
	UserName        string
	TriggerID       string
	CommandOrigin   command.Origin
	State           *slack.BlockActionStates
	ResponseURL     string
	BlockID         string
	EventTimeStamp  string
}
