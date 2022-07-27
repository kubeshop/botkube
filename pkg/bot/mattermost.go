package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	format2 "github.com/kubeshop/botkube/pkg/format"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// TODO: Refactor:
// 	- handle and send methods from `mattermostMessage` should be defined on Bot level,
//  - split to multiple files in a separate package,
//  - review all the methods and see if they can be simplified.

var _ Bot = &Mattermost{}

// mmChannelType to find Mattermost channel type
type mmChannelType string

const (
	mmChannelPrivate mmChannelType = "P"
	mmChannelPublic  mmChannelType = "O"
)

const (
	// WebSocketProtocol stores protocol initials for web socket
	WebSocketProtocol = "ws://"
	// WebSocketSecureProtocol stores protocol initials for web socket
	WebSocketSecureProtocol = "wss://"

	httpsScheme = "https"
)

// TODO:
// 	- Use latest Mattermost API v6
// 	- Remove usage of `log.Fatal` - return error instead

// Mattermost listens for user's message, execute commands and sends back the response.
type Mattermost struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	notify          bool

	Notification     config.Notification
	Token            string
	BotName          string
	TeamName         string
	ChannelID        string
	ClusterName      string
	AllowKubectl     bool
	RestrictAccess   bool
	ServerURL        string
	WebSocketURL     string
	WSClient         *model.WebSocketClient
	APIClient        *model.Client4
	DefaultNamespace string
}

// mattermostMessage contains message details to execute command and send back the result
type mattermostMessage struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	Event         *model.WebSocketEvent
	APIClient     *model.Client4
	Response      string
	Request       string
	IsAuthChannel bool
}

// NewMattermost creates a new Mattermost instance.
func NewMattermost(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Mattermost, error) {
	mattermost := c.Communications.GetFirst().Mattermost

	checkURL, err := url.Parse(mattermost.URL)
	if err != nil {
		return nil, fmt.Errorf("while parsing Mattermost URL %q: %w", mattermost.URL, err)
	}

	// Create WebSocketClient and handle messages
	webSocketURL := WebSocketProtocol + checkURL.Host
	if checkURL.Scheme == httpsScheme {
		webSocketURL = WebSocketSecureProtocol + checkURL.Host
	}

	client := model.NewAPIv4Client(mattermost.URL)
	client.SetOAuthToken(mattermost.Token)

	botTeam, resp := client.GetTeamByName(mattermost.Team, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	channel, resp := client.GetChannelByName(mattermost.Channels.GetFirst().Name, botTeam.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &Mattermost{
		log:              log,
		executorFactory:  executorFactory,
		reporter:         reporter,
		notify:           true, // enabled by default
		Notification:     mattermost.Notification,
		ServerURL:        mattermost.URL,
		BotName:          mattermost.BotName,
		Token:            mattermost.Token,
		TeamName:         mattermost.Team,
		ChannelID:        channel.Id,
		ClusterName:      c.Settings.ClusterName,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
		APIClient:        client,
		WebSocketURL:     webSocketURL,
	}, nil
}

// Start establishes mattermost connection and listens for messages
func (b *Mattermost) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	// Check connection to Mattermost server
	err := b.checkServerConnection()
	if err != nil {
		return fmt.Errorf("while pinging Mattermost server %q: %w", b.ServerURL, err)
	}

	err = b.reporter.ReportBotEnabled(b.IntegrationName())
	if err != nil {
		return fmt.Errorf("while reporting analytics: %w", err)
	}

	// It is observed that Mattermost server closes connections unexpectedly after some time.
	// For now, we are adding retry logic to reconnect to the server
	// https://github.com/kubeshop/botkube/issues/201
	b.log.Info("BotKube connected to Mattermost!")
	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return nil
		default:
			var appErr *model.AppError
			b.WSClient, appErr = model.NewWebSocketClient4(b.WebSocketURL, b.APIClient.AuthToken)
			if appErr != nil {
				return fmt.Errorf("while creating WebSocket connection: %w", appErr)
			}
			b.listen(ctx)
		}
	}
}

// IntegrationName describes the notifier integration name.
func (b *Mattermost) IntegrationName() config.CommPlatformIntegration {
	return config.MattermostCommPlatformIntegration
}

// Type describes the notifier type.
func (b *Mattermost) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// Enabled returns current notification status.
func (b *Mattermost) Enabled() bool {
	return b.notify
}

// SetEnabled sets a new notification status.
func (b *Mattermost) SetEnabled(value bool) error {
	b.notify = value
	return nil
}

// Check incoming message and take action
func (mm *mattermostMessage) handleMessage(b *Mattermost) {
	post := model.PostFromJson(strings.NewReader(mm.Event.Data["post"].(string)))
	channelType := mmChannelType(mm.Event.Data["channel_type"].(string))
	if channelType == mmChannelPrivate || channelType == mmChannelPublic {
		// Message posted in a channel
		// Serve only if starts with mention
		if !strings.HasPrefix(strings.ToLower(post.Message), fmt.Sprintf("@%s ", strings.ToLower(b.BotName))) {
			return
		}
	}

	// Check if message posted in authenticated channel
	if mm.Event.Broadcast.ChannelId == b.ChannelID {
		mm.IsAuthChannel = true
	}
	mm.log.Debugf("Received mattermost event: %+v", mm.Event.Data)

	// remove @BotKube prefix if exists
	r := regexp.MustCompile(`^(?i)@BotKube `)
	mm.Request = r.ReplaceAllString(post.Message, ``)

	e := mm.executorFactory.NewDefault(b.IntegrationName(), b, mm.IsAuthChannel, mm.Request)
	mm.Response = e.Execute()
	mm.sendMessage()
}

// Send messages to Mattermost
func (mm mattermostMessage) sendMessage() {
	mm.log.Debugf("Mattermost incoming Request: %s", mm.Request)
	mm.log.Debugf("Mattermost Response: %s", mm.Response)
	post := &model.Post{}
	post.ChannelId = mm.Event.Broadcast.ChannelId

	if len(mm.Response) == 0 {
		mm.log.Infof("Invalid request. Dumping the response. Request: %s", mm.Request)
		return
	}
	// Create file if message is too large
	if len(mm.Response) >= 3990 {
		res, resp := mm.APIClient.UploadFileAsRequestBody([]byte(mm.Response), mm.Event.Broadcast.ChannelId, mm.Request)
		if resp.Error != nil {
			mm.log.Error("Error occurred while uploading file. Error: ", resp.Error)
		}
		post.FileIds = []string{res.FileInfos[0].Id}
	} else {
		post.Message = format2.CodeBlock(mm.Response)
	}

	// Create a post in the ChannelName
	if _, resp := mm.APIClient.CreatePost(post); resp.Error != nil {
		mm.log.Error("Failed to send message. Error: ", resp.Error)
	}
}

// Check if Mattermost server is reachable
func (b *Mattermost) checkServerConnection() error {
	// Check api connection
	if _, resp := b.APIClient.GetOldClientConfig(""); resp.Error != nil {
		return resp.Error
	}

	// Get channel list
	_, resp := b.APIClient.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Check if team exists in Mattermost
func (b *Mattermost) getTeam() *model.Team {
	botTeam, resp := b.APIClient.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		b.log.Fatalf("There was a problem finding Mattermost team %s. %s", b.TeamName, resp.Error)
	}
	return botTeam
}

// Check if BotKube user exists in Mattermost
func (b *Mattermost) getUser() *model.User {
	users, resp := b.APIClient.AutocompleteUsersInTeam(b.getTeam().Id, b.BotName, 1, "")
	if resp.Error != nil {
		b.log.Fatalf("There was a problem finding Mattermost user %s. %s", b.BotName, resp.Error)
	}
	return users.Users[0]
}

func (b *Mattermost) listen(ctx context.Context) {
	b.WSClient.Listen()
	defer b.WSClient.Close()
	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return
		case event, ok := <-b.WSClient.EventChannel:
			if !ok {
				if b.WSClient.ListenError != nil {
					b.log.Debugf("while listening on websocket connection: %s", b.WSClient.ListenError.Error())
				}

				b.log.Info("Incoming events channel closed. Finishing...")
				return
			}

			if event == nil {
				b.log.Info("Nil event, ignoring")
				continue
			}

			if event.EventType() != model.WEBSOCKET_EVENT_POSTED {
				// ignore
				continue
			}

			post := model.PostFromJson(strings.NewReader(event.GetData()["post"].(string)))

			// Skip if message posted by BotKube or doesn't start with mention
			if post.UserId == b.getUser().Id {
				continue
			}
			mm := mattermostMessage{
				log:             b.log,
				executorFactory: b.executorFactory,
				Event:           event,
				IsAuthChannel:   false,
				APIClient:       b.APIClient,
			}
			mm.handleMessage(b)
		}
	}
}

// SendEvent sends event notification to Mattermost
func (b *Mattermost) SendEvent(ctx context.Context, event events.Event) error {
	if !b.notify {
		b.log.Info("Notifications are disabled. Skipping event...")
		return nil
	}

	b.log.Debugf(">> Sending to Mattermost: %+v", event)

	var fields []*model.SlackAttachmentField

	switch b.Notification.Type {
	case config.LongNotification:
		fields = b.longNotification(event)
	case config.ShortNotification:
		fallthrough

	default:
		// set missing cluster name to event object
		fields = b.shortNotification(event)
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
		targetChannel = b.ChannelID
	}
	isDefaultChannel := targetChannel == b.ChannelID

	post := &model.Post{
		Props: map[string]interface{}{
			"attachments": attachment,
		},
		ChannelId: targetChannel,
	}

	_, resp := b.APIClient.CreatePost(post)
	if resp.Error != nil {
		createPostWrappedErr := fmt.Errorf("while posting message to channel %q: %w", targetChannel, resp.Error)

		if isDefaultChannel {
			return createPostWrappedErr
		}

		// fallback to default channel

		// send error message to default channel
		msg := fmt.Sprintf("Unable to send message to ChannelName `%s`: `%s`\n```add Botkube app to the ChannelName %s\nMissed events follows below:```", targetChannel, resp.Error, targetChannel)
		sendMessageErr := b.SendMessage(ctx, msg)
		if sendMessageErr != nil {
			return multierror.Append(createPostWrappedErr, sendMessageErr)
		}

		// sending missed event to default channel
		// reset event.ChannelName and send event
		event.Channel = ""
		sendEventErr := b.SendEvent(ctx, event)
		if sendEventErr != nil {
			return multierror.Append(createPostWrappedErr, sendEventErr)
		}

		return createPostWrappedErr
	}

	b.log.Debugf("Event successfully sent to channel %q", post.ChannelId)
	return nil
}

// SendMessage sends message to Mattermost channel
func (b *Mattermost) SendMessage(_ context.Context, msg string) error {
	post := &model.Post{
		ChannelId: b.ChannelID,
		Message:   msg,
	}
	if _, resp := b.APIClient.CreatePost(post); resp.Error != nil {
		return fmt.Errorf("while creating a post: %w", resp.Error)
	}

	return nil
}
