package bot

import (
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/notify"
)

var themeColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "attention",
	config.Critical: "attention",
}

type fact map[string]interface{}

func formatTeamsMessage(event events.Event, notifType config.NotifType) map[string]interface{} {
	switch notifType {
	case config.LongNotify:
		return teamsLongNotification(event)

	case config.ShortNotify:
		fallthrough

	default:
		return teamsShortNotification(event)
	}
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

	sectionFacts := []fact{}

	if event.Cluster != "" {
		sectionFacts = append(sectionFacts, fact{
			"title": "Cluster",
			"value": event.Cluster,
		})
	}

	sectionFacts = append(sectionFacts, fact{
		"title": "Name",
		"value": event.Name,
	})

	if event.Namespace != "" {
		sectionFacts = append(sectionFacts, fact{
			"title": "Namespace",
			"value": event.Namespace,
		})
	}

	if event.Reason != "" {
		sectionFacts = append(sectionFacts, fact{
			"title": "Reason",
			"value": event.Reason,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message = message + m + "\n"
		}
		sectionFacts = append(sectionFacts, fact{
			"title": "Message",
			"value": message,
		})
	}

	if event.Action != "" {
		sectionFacts = append(sectionFacts, fact{
			"title": "Action",
			"value": event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec = rec + r + "\n"
		}
		sectionFacts = append(sectionFacts, fact{
			"title": "Recommendations",
			"value": rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn = warn + w + "\n"
		}
		sectionFacts = append(sectionFacts, fact{
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
