package notifier

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// customTimeFormat holds custom time format string
const customTimeFormat = "2006-01-02T15:04:05Z"

var embedColor = map[config.Level]int{
	config.Info:     8311585,  // green
	config.Warn:     16312092, // yellow
	config.Debug:    8311585,  // green
	config.Error:    13632027, // red
	config.Critical: 13632027, // red
}

// Discord contains URL and ClusterName
type Discord struct {
	log logrus.FieldLogger
	api *discordgo.Session

	Token        string
	ChannelID    string
	Notification config.Notification
}

// NewDiscord returns new Discord object
func NewDiscord(log logrus.FieldLogger, c config.Discord) (*Discord, error) {
	api, err := discordgo.New("Bot " + c.Token)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}

	return &Discord{
		log:          log,
		api:          api,
		ChannelID:    c.Channel,
		Notification: c.Notification,
	}, nil
}

// SendEvent sends event notification to Discord Channel
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752
func (d *Discord) SendEvent(_ context.Context, event events.Event) (err error) {
	d.log.Debugf(">> Sending to discord: %+v", event)

	messageSend := formatDiscordMessage(event, d.Notification)

	if _, err := d.api.ChannelMessageSendComplex(d.ChannelID, &messageSend); err != nil {
		return fmt.Errorf("while sending Discord message to channel %q: %w", d.ChannelID, err)
	}

	d.log.Debugf("Event successfully sent to channel %s", d.ChannelID)
	return nil
}

// SendMessage sends message to Discord Channel
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752
func (d *Discord) SendMessage(_ context.Context, msg string) error {
	d.log.Debugf(">> Sending to discord: %+v", msg)

	if _, err := d.api.ChannelMessageSend(d.ChannelID, msg); err != nil {
		return fmt.Errorf("while sending Discord message to channel %q: %w", d.ChannelID, err)
	}
	d.log.Debugf("Event successfully sent to Discord %v", msg)
	return nil
}

// IntegrationName describes the notifier integration name.
func (d *Discord) IntegrationName() config.CommPlatformIntegration {
	return config.DiscordCommPlatformIntegration
}

// Type describes the notifier type.
func (d *Discord) Type() config.IntegrationType {
	return config.BotIntegrationType
}
func formatDiscordMessage(event events.Event, notification config.Notification) discordgo.MessageSend {
	var messageEmbed discordgo.MessageEmbed

	switch notification.Type {
	case config.LongNotify:
		// generate Long notification message
		messageEmbed = discordLongNotification(event)

	case config.ShortNotify:
		// generate Short notification message
		fallthrough

	default:
		// generate Short notification message
		messageEmbed = discordShortNotification(event)
	}

	// Add timestamp
	messageEmbed.Timestamp = event.TimeStamp.UTC().Format(customTimeFormat)

	messageEmbed.Color = embedColor[event.Level]

	return discordgo.MessageSend{
		Embed: &messageEmbed,
	}
}

func discordLongNotification(event events.Event) discordgo.MessageEmbed {
	messageEmbed := discordgo.MessageEmbed{
		Title: fmt.Sprintf("*%s*", event.Title),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Kind",
				Value:  event.Kind,
				Inline: true,
			},
			{

				Name:   "Name",
				Value:  event.Name,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BotKube",
		},
	}
	if event.Namespace != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Namespace",
			Value:  event.Namespace,
			Inline: true,
		})
	}

	if event.Reason != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Reason",
			Value:  event.Reason,
			Inline: true,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message += fmt.Sprintf("%s\n", m)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Warnings",
			Value: warn,
		})
	}

	return messageEmbed
}

func discordShortNotification(event events.Event) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       event.Title,
		Description: FormatShortMessage(event),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BotKube",
		},
	}
}
