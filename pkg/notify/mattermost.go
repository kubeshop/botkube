package notify

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/mattermost/mattermost-server/model"
)

// Mattermost contains server URL and token
type Mattermost struct {
	Client      *model.Client4
	Channel     string
	ClusterName string
	NotifType   config.NotifType
}

// NewMattermost returns new Mattermost object
func NewMattermost(c *config.Config) (Notifier, error) {
	// Load configurations
	c, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	// Set configurations for Mattermost server
	client := model.NewAPIv4Client(c.Communications.Mattermost.URL)
	client.SetOAuthToken(c.Communications.Mattermost.Token)
	botTeam, resp := client.GetTeamByName(c.Communications.Mattermost.Team, "")
	if resp.Error != nil {
		log.Logger.Error("Error in connecting to Mattermost team ", c.Communications.Mattermost.Team, "\nError: ", resp.Error)
		return nil, resp.Error
	}
	botChannel, resp := client.GetChannelByName(c.Communications.Mattermost.Channel, botTeam.Id, "")
	if resp.Error != nil {
		log.Logger.Error("Error in connecting to Mattermost channel ", c.Communications.Mattermost.Channel, "\nError: ", resp.Error)
		return nil, resp.Error
	}

	return &Mattermost{
		Client:      client,
		Channel:     botChannel.Id,
		ClusterName: c.Settings.ClusterName,
		NotifType:   c.Communications.Mattermost.NotifType,
	}, nil
}

// SendEvent sends event notification to Mattermost
func (m *Mattermost) SendEvent(event events.Event) error {
	log.Logger.Info(fmt.Sprintf(">> Sending to Mattermost: %+v", event))

	var fields []*model.SlackAttachmentField

	switch m.NotifType {
	case config.LongNotify:
		fields = []*model.SlackAttachmentField{
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

		if event.Namespace != "" {
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Namespace",
				Value: event.Namespace,
				Short: true,
			})
		}

		if event.Reason != "" {
			fields = append(fields, &model.SlackAttachmentField{
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
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Message",
				Value: message,
			})
		}

		if event.Action != "" {
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Action",
				Value: event.Action,
			})
		}

		if len(event.Recommendations) > 0 {
			rec := ""
			for _, r := range event.Recommendations {
				rec = rec + r
			}
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Recommendations",
				Value: rec,
			})
		}

		if len(event.Warnings) > 0 {
			rec := ""
			for _, r := range event.Warnings {
				rec = rec + r
			}
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Warnings",
				Value: rec,
			})
		}

		// Add clustername in the message
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Cluster",
			Value: m.ClusterName,
		})

	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		event.Cluster = m.ClusterName
		fields = []*model.SlackAttachmentField{
			{
				Title: fmt.Sprintf("%s", event.Kind+" "+string(event.Type)),
				Value: event.Message(),
			},
		}
	}

	attachment := []*model.SlackAttachment{{
		Fields:    fields,
		Footer:    "BotKube",
		Timestamp: json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10)),
	}}

	post := &model.Post{}
	post.Props = map[string]interface{}{
		"attachments": attachment,
	}

	// non empty value in event.channel demands redirection of events to a different channel
	if event.Channel != "" {
		post.ChannelId = event.Channel

		if _, resp := m.Client.CreatePost(post); resp.Error != nil {
			log.Logger.Error("Failed to send message. Error: ", resp.Error)
			// send error message to default channel
			msg := fmt.Sprintf("Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```", event.Channel, resp.Error, event.Channel)
			go m.SendMessage(msg)
			// sending missed event to default channel
			// reset event.Channel and send event
			event.Channel = ""
			go m.SendEvent(event)
			return resp.Error
		}
		log.Logger.Debugf("Event successfully sent to channel %s", post.ChannelId)
	} else {
		post.ChannelId = m.Channel
		// empty value in event.channel sends notifications to default channel.
		if _, resp := m.Client.CreatePost(post); resp.Error != nil {
			log.Logger.Error("Failed to send message. Error: ", resp.Error)
			return resp.Error
		}
		log.Logger.Debugf("Event successfully sent to channel %s", post.ChannelId)
	}
	return nil
}

// SendMessage sends message to Mattermost channel
func (m *Mattermost) SendMessage(msg string) error {
	post := &model.Post{}
	post.ChannelId = m.Channel
	post.Message = msg
	if _, resp := m.Client.CreatePost(post); resp.Error != nil {
		log.Logger.Error("Failed to send message. Error: ", resp.Error)
	}
	return nil
}
