package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
)

// DiscordRenderer provides functionality to render Discord specific messages from a generic models.
type DiscordRenderer struct {
	mdFormatter interactive.MDFormatter
}

// NewDiscordRenderer returns new DiscordRenderer instance.
func NewDiscordRenderer() *DiscordRenderer {
	return &DiscordRenderer{
		mdFormatter: interactive.DefaultMDFormatter(),
	}
}

// MessageToMarkdown renders message in Markdown format.
func (d *DiscordRenderer) MessageToMarkdown(in interactive.CoreMessage) string {
	return interactive.RenderMessage(d.mdFormatter, in)
}

// NonInteractiveSectionToCard returns MessageEmbed for the given event message.
// Note: It cannot be used for other messages as we take into account only first message section with limited primitives:
// - TextFields
// - BulletLists
// - Timestamp
// It should be removed once we will add support for a proper message renderer.
func (d *DiscordRenderer) NonInteractiveSectionToCard(msg interactive.CoreMessage) (discordgo.MessageEmbed, error) {
	if err := IsValidNonInteractiveSingleSection(msg); err != nil {
		return discordgo.MessageEmbed{}, err
	}

	event := msg.Sections[0]
	messageEmbed := discordgo.MessageEmbed{
		Title:     event.Base.Header,
		Timestamp: d.renderTimestamp(msg.Timestamp),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Botkube",
		},
	}

	messageEmbed.Fields = append(messageEmbed.Fields, d.renderTextFields(event.TextFields)...)
	messageEmbed.Fields = append(messageEmbed.Fields, d.renderBulletLists(event.BulletLists)...)

	return messageEmbed, nil
}

func (*DiscordRenderer) renderTimestamp(in time.Time) string {
	if in.IsZero() {
		return ""
	}
	return in.UTC().Format("2006-01-02T15:04:05Z")
}

func (d *DiscordRenderer) renderTextFields(fields api.TextFields) []*discordgo.MessageEmbedField {
	var out []*discordgo.MessageEmbedField
	for _, field := range fields {
		if field.IsEmpty() {
			continue
		}
		out = append(out, &discordgo.MessageEmbedField{
			Name:   field.Key,
			Value:  field.Value,
			Inline: true,
		})
	}
	return out
}

func (d *DiscordRenderer) renderBulletLists(lists api.BulletLists) []*discordgo.MessageEmbedField {
	var out []*discordgo.MessageEmbedField
	for _, item := range lists {
		out = append(out, &discordgo.MessageEmbedField{
			Name:   item.Title,
			Value:  formatx.BulletPointListFromMessages(item.Items),
			Inline: false,
		})
	}
	return out
}
