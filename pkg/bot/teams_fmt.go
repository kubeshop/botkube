package bot

import (
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/format"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

var themeColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "attention",
	config.Critical: "attention",
}

// TODO: Use dedicated types as a part of https://github.com/kubeshop/botkube/issues/667
type fact map[string]interface{}

func (b *Teams) formatMessage(event event.Event, notification config.Notification) map[string]interface{} {
	switch notification.Type {
	case config.LongNotification:
		return b.longNotification(event)

	case config.ShortNotification:
		fallthrough

	default:
		return b.shortNotification(event)
	}
}

func (b *Teams) shortNotification(event event.Event) map[string]interface{} {
	return map[string]interface{}{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.0",
		"body": []map[string]interface{}{
			{
				"type":  "TextBlock",
				"text":  event.Title,
				"size":  "Large",
				"color": themeColor[event.Level],
				"wrap":  true,
			},
			{
				"type": "TextBlock",
				"text": strings.ReplaceAll(format.ShortMessage(event), "```", ""),
				"wrap": true,
			},
		},
	}
}

func (b *Teams) longNotification(event event.Event) map[string]interface{} {
	// TODO: Use dedicated types as a part of https://github.com/kubeshop/botkube/issues/667
	card := map[string]interface{}{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.0",
	}

	sectionFacts := []fact{
		{
			"title": "Kind",
			"value": event.Kind,
		},
		{
			"title": "Name",
			"value": event.Name,
		},
	}

	sectionFacts = b.appendIfNotEmpty(sectionFacts, event.Namespace, "Namespace")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, event.Reason, "Reason")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, formatx.JoinMessages(event.Messages), "Message")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, event.Action, "Action")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, formatx.JoinMessages(event.Recommendations), "Recommendations")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, formatx.JoinMessages(event.Warnings), "Warnings")
	sectionFacts = b.appendIfNotEmpty(sectionFacts, event.Cluster, "Cluster")

	card["body"] = []map[string]interface{}{
		{
			"type":  "TextBlock",
			"text":  event.Title,
			"size":  "Large",
			"color": themeColor[event.Level],
		},
		{
			"type":  "FactSet",
			"facts": sectionFacts,
		},
	}
	return card
}

func (b *Teams) appendIfNotEmpty(fields []fact, in string, title string) []fact {
	if in == "" {
		return fields
	}
	return append(fields, fact{
		"title": title,
		"value": in,
	})
}
