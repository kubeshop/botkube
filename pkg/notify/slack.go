package notify

import (
	"fmt"

	"github.com/infracloudio/kubeops/pkg/config"
	"github.com/infracloudio/kubeops/pkg/events"
	log "github.com/infracloudio/kubeops/pkg/logging"
	"github.com/nlopes/slack"
)

var AttachmentColor map[events.Level]string

type SlackMessage struct {
}

type Slack struct {
	Token   string
	Channel string
}

func NewSlack() Notifier {
	AttachmentColor = map[events.Level]string{
		events.Info:     "#00ff00",
		events.Warn:     "#ffff00",
		events.Debug:    "#00ff00",
		events.Error:    "#ff0000",
		events.Critical: "#ff0000",
	}

	c, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	return &Slack{
		Token:   c.Communications.Slack.Token,
		Channel: c.Communications.Slack.Channel,
	}
}

func (s *Slack) Send(event events.Event) error {
	log.Logger.Info(fmt.Sprintf(">> Sending to slack: %+v", event))

	api := slack.New(s.Token)
	params := slack.PostMessageParameters{}
	attachment := slack.Attachment{
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "Kind",
				Value: event.Kind,
				Short: true,
			},
			slack.AttachmentField{
				Title: "Name",
				Value: event.Name,
				Short: true,
			},
			slack.AttachmentField{
				Title: "Namespace",
				Value: event.Namespace,
				Short: true,
			},
		},
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

	if event.Reason != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Reason",
			Value: event.Reason,
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

	attachment.Color = AttachmentColor[event.Level]
	params.Attachments = []slack.Attachment{attachment}

	log.Logger.Infof("Sending message on %v with token %s", s.Channel, s.Token)
	channelID, timestamp, err := api.PostMessage(s.Channel, "", params)
	if err != nil {
		log.Logger.Errorf("Error in sending slack message %s", err.Error())
		return err
	}

	log.Logger.Infof("Message successfully sent to channel %s at %s", channelID, timestamp)
	return nil
}
