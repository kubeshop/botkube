package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

func (b *Discord) formatMessage(event events.Event) discordgo.MessageSend {
	var messageEmbed discordgo.MessageEmbed

	switch b.notification.Type {
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
			Text: "Botkube",
		},
	}

	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, event.Namespace, "Namespace", true)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, event.Reason, "Reason", true)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, formatx.JoinMessages(event.Messages), "Message", false)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, event.Action, "Action", true)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, formatx.JoinMessages(event.Recommendations), "Recommendations", false)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, formatx.JoinMessages(event.Warnings), "Warnings", false)
	messageEmbed.Fields = b.appendIfNotEmpty(messageEmbed.Fields, event.Cluster, "Cluster", false)

	return messageEmbed
}

func (b *Discord) appendIfNotEmpty(fields []*discordgo.MessageEmbedField, in string, title string, short bool) []*discordgo.MessageEmbedField {
	if in == "" {
		return fields
	}
	return append(fields, &discordgo.MessageEmbedField{
		Name:   title,
		Value:  in,
		Inline: short,
	})
}

func (b *Discord) shortNotification(event events.Event) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       event.Title,
		Description: formatx.ShortMessage(event),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Botkube",
		},
	}
}
