package bot

import (
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/notify"
)

const (
	// Constants for sending  messageCards
	messageType   = "MessageCard"
	schemaContext = "http://schema.org/extensions"
)

var themeColor = map[events.Level]string{
	events.Info:     "good",
	events.Warn:     "warning",
	events.Debug:    "good",
	events.Error:    "attention",
	events.Critical: "attention",
}

type Fact map[string]interface{}

func formatTeamsMessage(event events.Event, notifType config.NotifType) map[string]interface{} {
	switch notifType {
	case config.LongNotify:
		return teamsLongNotification(event)

	case config.ShortNotify:
		fallthrough

	default:
		return teamsShortNotification(event)
	}
	return nil
}

func teamsShortNotification(event events.Event) map[string]interface{} {
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
				"text": strings.ReplaceAll(notify.FormatShortMessage(event), "```", ""),
				"wrap": true,
			},
		},
	}
}

func teamsLongNotification(event events.Event) map[string]interface{} {
	card := map[string]interface{}{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.0",
	}

	sectionFacts := []Fact{}

	if event.Cluster != "" {
		sectionFacts = append(sectionFacts, Fact{
			"title": "Cluster",
			"value": event.Cluster,
		})
	}

	sectionFacts = append(sectionFacts, Fact{
		"title": "Name",
		"value": event.Name,
	})

	if event.Namespace != "" {
		sectionFacts = append(sectionFacts, Fact{
			"title": "Namespace",
			"value": event.Namespace,
		})
	}

	if event.Reason != "" {
		sectionFacts = append(sectionFacts, Fact{
			"title": "Reason",
			"value": event.Reason,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message = message + m + "\n"
		}
		sectionFacts = append(sectionFacts, Fact{
			"title": "Message",
			"value": message,
		})
	}

	if event.Action != "" {
		sectionFacts = append(sectionFacts, Fact{
			"title": "Action",
			"value": event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec = rec + r + "\n"
		}
		sectionFacts = append(sectionFacts, Fact{
			"title": "Recommendations",
			"value": rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn = warn + w + "\n"
		}
		sectionFacts = append(sectionFacts, Fact{
			"title": "Warnings",
			"value": warn,
		})
	}

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
