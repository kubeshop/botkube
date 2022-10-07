package bot

import (
	"encoding/json"
	"strconv"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

func (b *Mattermost) formatAttachments(event events.Event) []*model.SlackAttachment {
	var fields []*model.SlackAttachmentField
	switch b.notification.Type {
	case config.LongNotification:
		fields = b.longNotification(event)
	case config.ShortNotification:
		fallthrough
	default:
		// set missing cluster name to the event object
		fields = b.shortNotification(event)
	}

	return []*model.SlackAttachment{
		{
			Color:     attachmentColor[event.Level],
			Title:     event.Title,
			Fields:    fields,
			Footer:    "Botkube",
			Timestamp: json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10)),
		},
	}
}

func (b *Mattermost) longNotification(event events.Event) []*model.SlackAttachmentField {
	fields := []*model.SlackAttachmentField{
		{
			Title: "Kind",
			Value: event.Kind,
			Short: true,
		},
		{
			Title: "Name",
			Value: event.Name,
			Short: true,
		},
	}

	fields = b.appendIfNotEmpty(fields, event.Namespace, "Namespace", true)
	fields = b.appendIfNotEmpty(fields, event.Reason, "Reason", true)
	fields = b.appendIfNotEmpty(fields, formatx.JoinMessages(event.Messages), "Message", false)
	fields = b.appendIfNotEmpty(fields, event.Action, "Action", true)
	fields = b.appendIfNotEmpty(fields, formatx.JoinMessages(event.Recommendations), "Recommendations", false)
	fields = b.appendIfNotEmpty(fields, formatx.JoinMessages(event.Warnings), "Warnings", false)
	fields = b.appendIfNotEmpty(fields, event.Cluster, "Cluster", false)

	return fields
}

func (b *Mattermost) appendIfNotEmpty(fields []*model.SlackAttachmentField, in string, title string, short model.SlackCompatibleBool) []*model.SlackAttachmentField {
	if in == "" {
		return fields
	}
	return append(fields, &model.SlackAttachmentField{
		Title: title,
		Value: in,
		Short: short,
	})
}

func (b *Mattermost) shortNotification(event events.Event) []*model.SlackAttachmentField {
	return []*model.SlackAttachmentField{
		{
			Value: formatx.ShortMessage(event),
		},
	}
}
