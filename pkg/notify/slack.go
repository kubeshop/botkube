package notify

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/nlopes/slack"
)

var attachmentColor map[events.Level]string

// Slack contains Token for authentication with slack and Channel name to send notification to
type Slack struct {
	Token       string
	Channel     string
	ClusterName string
	NotifType   config.NotifType
}

// NewSlack returns new Slack object
func NewSlack(c *config.Config) Notifier {
	attachmentColor = map[events.Level]string{
		events.Info:     "good",
		events.Warn:     "warning",
		events.Debug:    "good",
		events.Error:    "danger",
		events.Critical: "danger",
	}

	return &Slack{
		Token:       c.Communications.Slack.Token,
		Channel:     c.Communications.Slack.Channel,
		ClusterName: c.Settings.ClusterName,
		NotifType:   c.Communications.Slack.NotifType,
	}
}

// SendEvent sends event notification to slack
func (s *Slack) SendEvent(event events.Event) error {
	log.Logger.Debug(fmt.Sprintf(">> Sending to slack: %+v", event))

	api := slack.New(s.Token)

	var attachment slack.Attachment

	switch s.NotifType {
	case config.LongNotify:
		attachment = slack.Attachment{
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
				message = message + m
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
				rec = rec + r
			}
			attachment.Fields = append(attachment.Fields, slack.AttachmentField{
				Title: "Recommendations",
				Value: rec,
			})
		}

		// Add clustername in the message
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Cluster",
			Value: s.ClusterName,
		})

	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		event.Cluster = s.ClusterName
		attachment = slack.Attachment{
			Fields: []slack.AttachmentField{
				{
					Title: fmt.Sprintf("%s", event.Kind+" "+string(event.Type)),
					Value: event.Message(),
				},
			},
			Footer: "BotKube",
		}
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]

	// non empty value in event.channel demands redirection of events to a different channel
	if event.Channel != "" {
		channelID, timestamp, err := api.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Logger.Errorf("Error in sending slack message %s", err.Error())
			// send error message to default channel
			if err.Error() == "channel_not_found" {
				msg := fmt.Sprintf("Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```", event.Channel, err.Error(), event.Channel)
				go s.SendMessage(msg)
				// sending missed event to default channel
				// reset event.Channel and send event
				event.Channel = ""
				go s.SendEvent(event)
			}
			return err
		}
		log.Logger.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	} else {
		// empty value in event.channel sends notifications to default channel.
		channelID, timestamp, err := api.PostMessage(s.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Logger.Errorf("Error in sending slack message %s", err.Error())
			return err
		}
		log.Logger.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	}
	return nil
}

// SendMessage sends message to slack channel
func (s *Slack) SendMessage(msg string) error {
	log.Logger.Debug(fmt.Sprintf(">> Sending to slack: %+v", msg))

	api := slack.New(s.Token)

	channelID, timestamp, err := api.PostMessage(s.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		log.Logger.Errorf("Error in sending slack message %s", err.Error())
		return err
	}

	log.Logger.Debugf("Message successfully sent to channel %s at %s", channelID, timestamp)
	return nil
}
