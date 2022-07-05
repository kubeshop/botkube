package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
)

// Mattermost contains server URL and token
type Mattermost struct {
	log       logrus.FieldLogger
	Client    *model.Client4
	Channel   string
	NotifType config.NotifType
}

// NewMattermost returns new Mattermost object
func NewMattermost(log logrus.FieldLogger, c config.Mattermost) (*Mattermost, error) {
	// Set configurations for Mattermost server
	client := model.NewAPIv4Client(c.URL)
	client.SetOAuthToken(c.Token)
	botTeam, resp := client.GetTeamByName(c.Team, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	botChannel, resp := client.GetChannelByName(c.Channel, botTeam.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &Mattermost{
		log:       log,
		Client:    client,
		Channel:   botChannel.Id,
		NotifType: c.NotifType,
	}, nil
}

// SendEvent sends event notification to Mattermost
func (m *Mattermost) SendEvent(ctx context.Context, event events.Event) error {
	m.log.Debugf(">> Sending to Mattermost: %+v", event)

	var fields []*model.SlackAttachmentField

	switch m.NotifType {
	case config.LongNotify:
		fields = mmLongNotification(event)
	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		fields = mmShortNotification(event)
	}

	attachment := []*model.SlackAttachment{
		{
			Color:     attachmentColor[event.Level],
			Title:     event.Title,
			Fields:    fields,
			Footer:    "BotKube",
			Timestamp: json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10)),
		},
	}

	targetChannel := event.Channel
	if targetChannel == "" {
		// empty value in event.channel sends notifications to default channel.
		targetChannel = m.Channel
	}
	isDefaultChannel := targetChannel == m.Channel

	post := &model.Post{
		Props: map[string]interface{}{
			"attachments": attachment,
		},
		ChannelId: targetChannel,
	}

	_, resp := m.Client.CreatePost(post)
	if resp.Error != nil {
		createPostWrappedErr := fmt.Errorf("while posting message to channel %q: %w", targetChannel, resp.Error)

		if isDefaultChannel {
			return createPostWrappedErr
		}

		// fallback to default channel

		// send error message to default channel
		msg := fmt.Sprintf("Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```", targetChannel, resp.Error, targetChannel)
		sendMessageErr := m.SendMessage(ctx, msg)
		if sendMessageErr != nil {
			return multierror.Append(createPostWrappedErr, sendMessageErr)
		}

		// sending missed event to default channel
		// reset event.Channel and send event
		event.Channel = ""
		sendEventErr := m.SendEvent(ctx, event)
		if sendEventErr != nil {
			return multierror.Append(createPostWrappedErr, sendEventErr)
		}

		return createPostWrappedErr
	}

	m.log.Debugf("Event successfully sent to channel %q", post.ChannelId)
	return nil
}

// SendMessage sends message to Mattermost channel
func (m *Mattermost) SendMessage(_ context.Context, msg string) error {
	post := &model.Post{
		ChannelId: m.Channel,
		Message:   msg,
	}
	if _, resp := m.Client.CreatePost(post); resp.Error != nil {
		return fmt.Errorf("while creating a post: %w", resp.Error)
	}

	return nil
}

func mmLongNotification(event events.Event) []*model.SlackAttachmentField {
	fields := []*model.SlackAttachmentField{
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
			message += fmt.Sprintf("%s\n", m)
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
			rec += fmt.Sprintf("%s\n", r)
		}
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}

		fields = append(fields, &model.SlackAttachmentField{
			Title: "Warnings",
			Value: warn,
		})
	}

	// Add clustername in the message
	fields = append(fields, &model.SlackAttachmentField{
		Title: "Cluster",
		Value: event.Cluster,
	})
	return fields
}

func mmShortNotification(event events.Event) []*model.SlackAttachmentField {
	return []*model.SlackAttachmentField{
		{
			Value: FormatShortMessage(event),
		},
	}
}
