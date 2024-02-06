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

func slackError(err error, channel string) error {
	switch err.Error() {
	case "channel_not_found":
		err = fmt.Errorf("channel %q not found", channel)
	case "not_in_channel":
		err = fmt.Errorf("botkube is not in channel %q", channel)
	case "invalid_auth":
		err = fmt.Errorf("invalid slack credentials")
	}
	return err
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	Text                 string
	Channel              string
	ThreadTimeStamp      string
	UserID               string
	UserName             string
	TriggerID            string
	CommandOrigin        command.Origin
	State                *slack.BlockActionStates
	ResponseURL          string
	BlockID              string
	EventTimeStamp       string
	RootMessageTimeStamp string
}

// GetTimestamp returns the timestamp for the response message.
func (s *slackMessage) GetTimestamp() string {
	// If the event is coming from the thread, then we simply respond in that thread
	if s.ThreadTimeStamp != "" {
		return s.ThreadTimeStamp
	}
	// otherwise, we use the event timestamp to respond in the thread to the message that triggered our response
	return s.EventTimeStamp
}
