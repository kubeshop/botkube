package format

import (
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

const bulletPointFmt = "- %s\n"

// ShortMessage prepares message in short event format.
func ShortMessage(event events.Event) string {
	msg := messageHeader(event)
	msgAttachments := messageAttachments(event)

	return fmt.Sprintf("%s\n%s", msg, msgAttachments)
}

func messageHeader(event events.Event) string {
	resourceName := event.Name
	if event.Namespace != "" {
		resourceName = fmt.Sprintf("%s/%s", event.Namespace, event.Name)
	}

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		return fmt.Sprintf(
			"%s *%s* has been %s in *%s* cluster",
			event.Kind,
			resourceName,
			event.Type+"d",
			event.Cluster,
		)

	case config.ErrorEvent:
		return fmt.Sprintf(
			"Error occurred for %s *%s* in *%s* cluster",
			event.Kind,
			resourceName,
			event.Cluster,
		)

	case config.WarningEvent:
		return fmt.Sprintf(
			"Warning for %s *%s* in *%s* cluster",
			event.Kind,
			resourceName,
			event.Cluster,
		)

	case config.InfoEvent, config.NormalEvent:
		return fmt.Sprintf(
			"Info for %s *%s* in *%s* cluster",
			event.Kind,
			resourceName,
			event.Cluster,
		)
	}

	return ""
}

func messageAttachments(event events.Event) string {
	var additionalMsgStrBuilder strings.Builder
	if len(event.Messages) > 0 {
		for _, m := range event.Messages {
			additionalMsgStrBuilder.WriteString(fmt.Sprintf("%s\n", m))
		}
	}
	if len(event.Recommendations) > 0 {
		additionalMsgStrBuilder.WriteString("Recommendations:\n")

		for _, m := range event.Recommendations {
			additionalMsgStrBuilder.WriteString(fmt.Sprintf(bulletPointFmt, m))
		}
	}
	if len(event.Warnings) > 0 {
		additionalMsgStrBuilder.WriteString("Warnings:\n")

		for _, m := range event.Warnings {
			additionalMsgStrBuilder.WriteString(fmt.Sprintf(bulletPointFmt, m))
		}
	}

	if additionalMsgStrBuilder.Len() == 0 {
		return ""
	}

	return fmt.Sprintf("```\n%s```", additionalMsgStrBuilder.String())
}
