package bot

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
)

// MattermostRenderer provides functionality to render Mattermost specific messages from a generic models.
type MattermostRenderer struct {
	mdFormatter interactive.MDFormatter
}

// NewMattermostRenderer returns new MattermostRenderer instance.
func NewMattermostRenderer() *MattermostRenderer {
	return &MattermostRenderer{
		mdFormatter: interactive.DefaultMDFormatter(),
	}
}

// MessageToMarkdown renders message in Markdown format.
func (d *MattermostRenderer) MessageToMarkdown(in interactive.CoreMessage) string {
	return interactive.RenderMessage(d.mdFormatter, in)
}

// NonInteractiveSectionToCard returns MessageEmbed for the given event message.
// Note: It cannot be used for other messages as we take into account only first message section with limited primitives:
// - TextFields
// - BulletLists
// - Timestamp
// It should be removed once we will add support for a proper message renderer.
func (d *MattermostRenderer) NonInteractiveSectionToCard(msg interactive.CoreMessage) ([]*model.SlackAttachment, error) {
	if err := IsValidNonInteractiveSingleSection(msg); err != nil {
		return nil, err
	}

	event := msg.Sections[0]
	messageAttachment := &model.SlackAttachment{
		Title:     event.Base.Header,
		Timestamp: d.renderTimestamp(msg.Timestamp),
		Footer:    "Botkube",
	}

	messageAttachment.Fields = append(messageAttachment.Fields, d.renderTextFields(event.TextFields)...)
	messageAttachment.Fields = append(messageAttachment.Fields, d.renderBulletLists(event.BulletLists)...)

	return []*model.SlackAttachment{
		messageAttachment,
	}, nil
}

func (*MattermostRenderer) renderTimestamp(in time.Time) json.Number {
	if in.IsZero() {
		return ""
	}
	return json.Number(strconv.FormatInt(in.Unix(), 10))
}

func (d *MattermostRenderer) renderTextFields(fields api.TextFields) []*model.SlackAttachmentField {
	var out []*model.SlackAttachmentField
	for _, field := range fields {
		if field.IsEmpty() {
			continue
		}
		out = append(out, &model.SlackAttachmentField{
			Title: field.Key,
			Value: field.Value,
			Short: true,
		})
	}
	return out
}

func (d *MattermostRenderer) renderBulletLists(lists api.BulletLists) []*model.SlackAttachmentField {
	var out []*model.SlackAttachmentField
	for _, item := range lists {
		out = append(out, &model.SlackAttachmentField{
			Title: item.Title,
			Value: formatx.BulletPointListFromMessages(item.Items),
			Short: false,
		})
	}
	return out
}
