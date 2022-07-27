package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/format"
)

func (b *Discord) formatMessage(event events.Event, notification config.Notification) discordgo.MessageSend {
	var messageEmbed discordgo.MessageEmbed

	switch notification.Type {
	case config.LongNotification:
		// generate Long notification message
		messageEmbed = b.longNotification(event)

	case config.ShortNotification:
		// generate Short notification message
		fallthrough

	default:
		// generate Short notification message
		messageEmbed = b.shortNotification(event)
	}

	// Add timestamp
	messageEmbed.Timestamp = event.TimeStamp.UTC().Format(customTimeFormat)

	messageEmbed.Color = embedColor[event.Level]

	return discordgo.MessageSend{
		Embed: &messageEmbed,
	}
}

func (b *Discord) longNotification(event events.Event) discordgo.MessageEmbed {
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

func (b *Discord) shortNotification(event events.Event) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       event.Title,
		Description: format.ShortMessage(event),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BotKube",
		},
	}
}
