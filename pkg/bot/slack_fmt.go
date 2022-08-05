package bot

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

func (b *Slack) formatMessage(event events.Event) slack.Attachment {
	var attachment slack.Attachment

	switch b.notification.Type {
	case config.LongNotification:
		attachment = b.longNotification(event)
	case config.ShortNotification:
		fallthrough
	default:
		attachment = b.shortNotification(event)
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]
	return attachment
}

func (b *Slack) longNotification(event events.Event) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*%s*", event.Title),
		Fields: []slack.AttachmentField{
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
		},
		Footer: "BotKube",
	}

	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Namespace, "Namespace", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Reason, "Reason", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Messages), "Message", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Action, "Action", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Recommendations), "Recommendations", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Warnings), "Warnings", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Cluster, "Cluster", false)

	return attachment
}

func (b *Slack) appendIfNotEmpty(fields []slack.AttachmentField, in string, title string, short bool) []slack.AttachmentField {
	if in == "" {
		return fields
	}
	return append(fields, slack.AttachmentField{
		Title: title,
		Value: in,
		Short: short,
	})
}

func (b *Slack) shortNotification(event events.Event) slack.Attachment {
	return slack.Attachment{
		Title: event.Title,
		Fields: []slack.AttachmentField{
			{
				Value: formatx.ShortMessage(event),
			},
		},
		Footer: "BotKube",
	}
}
