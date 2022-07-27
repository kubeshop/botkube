package format

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// ShortMessage prepares message in short event format.
func ShortMessage(event events.Event) (msg string) {
	additionalMsg := ""
	if len(event.Messages) > 0 {
		for _, m := range event.Messages {
			additionalMsg += fmt.Sprintf("%s\n", m)
		}
	}
	if len(event.Recommendations) > 0 {
		recommend := ""
		for _, m := range event.Recommendations {
			recommend += fmt.Sprintf("- %s\n", m)
		}
		additionalMsg += fmt.Sprintf("Recommendations:\n%s", recommend)
	}
	if len(event.Warnings) > 0 {
		warning := ""
		for _, m := range event.Warnings {
			warning += fmt.Sprintf("- %s\n", m)
		}
		additionalMsg += fmt.Sprintf("Warnings:\n%s", warning)
	}

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"%s *%s* has been %s in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"%s *%s/%s* has been %s in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		}
	case config.ErrorEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	case config.WarningEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Warning %s: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Warning %s: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	case config.InfoEvent, config.NormalEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"%s Info: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"%s Info: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	}

	// Add message in the attachment if there is any
	if len(additionalMsg) > 0 {
		msg += fmt.Sprintf("```\n%s```", additionalMsg)
	}
	return msg
}
