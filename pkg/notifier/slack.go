package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

const sendFailureMessageFmt = "Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```"
const channelNotFoundCode = "channel_not_found"

var attachmentColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "danger",
	config.Critical: "danger",
}

// Slack contains Token for authentication with slack and Channel name to send notification to
type Slack struct {
	log logrus.FieldLogger

	Channel      string
	Notification config.Notification
	Client       *slack.Client
}

// NewSlack returns new Slack object
func NewSlack(log logrus.FieldLogger, c config.Slack) *Slack {
	return &Slack{
		log:          log,
		Channel:      c.Channel,
		Notification: c.Notification,
		Client:       slack.New(c.Token),
	}
}

// SendEvent sends event notification to slack
func (s *Slack) SendEvent(ctx context.Context, event events.Event) error {
	s.log.Debugf(">> Sending to slack: %+v", event)
	attachment := formatSlackMessage(event, s.Notification)

	targetChannel := event.Channel
	if targetChannel == "" {
		// empty value in event.channel sends notifications to default channel.
		targetChannel = s.Channel
	}
	isDefaultChannel := targetChannel == s.Channel

	channelID, timestamp, err := s.Client.PostMessage(targetChannel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
	if err != nil {
		postMessageWrappedErr := fmt.Errorf("while posting message to channel %q: %w", targetChannel, err)

		if isDefaultChannel || err.Error() != channelNotFoundCode {
			return postMessageWrappedErr
		}

		// channel not found, fallback to default channel

		// send error message to default channel
		msg := fmt.Sprintf(sendFailureMessageFmt, targetChannel, err.Error(), targetChannel)
		sendMessageErr := s.SendMessage(ctx, msg)
		if sendMessageErr != nil {
			return multierror.Append(postMessageWrappedErr, sendMessageErr)
		}

		// sending missed event to default channel
		// reset event.Channel and send event
		event.Channel = ""
		sendEventErr := s.SendEvent(ctx, event)
		if sendEventErr != nil {
			return multierror.Append(postMessageWrappedErr, sendEventErr)
		}

		return postMessageWrappedErr
	}

	s.log.Debugf("Event successfully sent to channel %q at %s", channelID, timestamp)
	return nil
}

// SendMessage sends message to slack channel
func (s *Slack) SendMessage(ctx context.Context, msg string) error {
	s.log.Debugf(">> Sending to slack: %+v", msg)
	channelID, timestamp, err := s.Client.PostMessageContext(ctx, s.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		return fmt.Errorf("while sending Slack message to channel %q: %w", s.Channel, err)
	}

	s.log.Debugf("Message successfully sent to channel %s at %s", channelID, timestamp)
	return nil
}

func formatSlackMessage(event events.Event, notification config.Notification) (attachment slack.Attachment) {
	switch notification.Type {
	case config.LongNotify:
		attachment = slackLongNotification(event)

	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		attachment = slackShortNotification(event)
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]
	return attachment
}

func slackLongNotification(event events.Event) slack.Attachment {
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
	if event.Namespace != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Namespace",
			Value: event.Namespace,
			Short: true,
		})
	}

	if event.Reason != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Reason",
			Value: event.Reason,
			Short: true,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message += fmt.Sprintf("%s\n", m)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Warnings",
			Value: warn,
		})
	}

	// Add clusterName in the message
	attachment.Fields = append(attachment.Fields, slack.AttachmentField{
		Title: "Cluster",
		Value: event.Cluster,
	})
	return attachment
}

func slackShortNotification(event events.Event) slack.Attachment {
	return slack.Attachment{
		Title: event.Title,
		Fields: []slack.AttachmentField{
			{
				Value: FormatShortMessage(event),
			},
		},
		Footer: "BotKube",
	}
}

// FormatShortMessage prepares message in short event format
func FormatShortMessage(event events.Event) (msg string) {
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

// IntegrationName describes the notifier integration name.
func (s *Slack) IntegrationName() config.CommPlatformIntegration {
	return config.SlackCommPlatformIntegration
}

// Type describes the notifier type.
func (s *Slack) Type() config.IntegrationType {
	return config.BotIntegrationType
}
