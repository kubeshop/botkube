package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

var _ Bot = &MattermostBot{}

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
)

// TODO:
// 	- Use latest Mattermost API v6
// 	- Remove usage of `log.Fatal` - return error instead

// MattermostBot listens for user's message, execute commands and sends back the response
type MattermostBot struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	notify          bool

	Notification     config.Notification
	Token            string
	BotName          string
	TeamName         string
	ChannelName      string
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
	Response      string
	Request       string
	IsAuthChannel bool
	APIClient     *model.Client4
}

// NewMattermostBot returns new Bot object
func NewMattermostBot(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*MattermostBot, error) {
	mattermost := c.Communications.GetFirst().Mattermost

	// Set configurations for MattermostBot server
	client := model.NewAPIv4Client(mattermost.URL)
	client.SetOAuthToken(mattermost.Token)
	botTeam, resp := client.GetTeamByName(mattermost.Team, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	botChannel, resp := client.GetChannelByName(mattermost.Channels.GetFirst().Name, botTeam.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &MattermostBot{
		log:              log,
		executorFactory:  executorFactory,
		reporter:         reporter,
		notify:           true, // enabled by default
		Notification:     mattermost.Notification,
		ServerURL:        mattermost.URL,
		BotName:          mattermost.BotName,
		Token:            mattermost.Token,
		TeamName:         mattermost.Team,
		ChannelName:      mattermost.Channels.GetFirst().Name,
		ClusterName:      c.Settings.ClusterName,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
		APIClient:        client,
	}
}

// Start establishes mattermost connection and listens for messages
func (b *MattermostBot) Start(ctx context.Context) error {
	b.log.Info("Starting bot")
	b.APIClient = model.NewAPIv4Client(b.ServerURL)
	b.APIClient.SetOAuthToken(b.Token)

	// Check if Mattermost URL is valid
	checkURL, err := url.Parse(b.ServerURL)
	if err != nil {
		return fmt.Errorf("while parsing Mattermost URL %q: %w", b.ServerURL, err)
	}

	// Create WebSocketClient and handle messages
	b.WebSocketURL = WebSocketProtocol + checkURL.Host
	if checkURL.Scheme == "https" {
		b.WebSocketURL = WebSocketSecureProtocol + checkURL.Host
	}

	// Check connection to Mattermost server
	err = b.checkServerConnection()
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

// IntegrationName describes the sink integration name.
func (b *MattermostBot) IntegrationName() config.CommPlatformIntegration {
	return config.MattermostCommPlatformIntegration
}

// Type describes the sink type.
func (b *MattermostBot) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// Enabled returns current notification status.
func (b *MattermostBot) Enabled() bool {
	return b.notify
}

// SetEnabled sets a new notification status.
func (b *MattermostBot) SetEnabled(value bool) error {
	b.notify = value
	return nil
}

// TODO: refactor - handle and send methods should be defined on Bot level

// Check incoming message and take action
func (mm *mattermostMessage) handleMessage(b *MattermostBot) {
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
	if mm.Event.Broadcast.ChannelId == b.getChannel().Id {
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
		post.Message = formatCodeBlock(mm.Response)
	}

	// Create a post in the ChannelName
	if _, resp := mm.APIClient.CreatePost(post); resp.Error != nil {
		mm.log.Error("Failed to send message. Error: ", resp.Error)
	}
}

// Check if Mattermost server is reachable
func (b *MattermostBot) checkServerConnection() error {
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
func (b *MattermostBot) getTeam() *model.Team {
	botTeam, resp := b.APIClient.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		b.log.Fatalf("There was a problem finding Mattermost team %s. %s", b.TeamName, resp.Error)
	}
	return botTeam
}

// Check if BotKube user exists in Mattermost
func (b *MattermostBot) getUser() *model.User {
	users, resp := b.APIClient.AutocompleteUsersInTeam(b.getTeam().Id, b.BotName, 1, "")
	if resp.Error != nil {
		b.log.Fatalf("There was a problem finding Mattermost user %s. %s", b.BotName, resp.Error)
	}
	return users.Users[0]
}

// Create channel if not present and add BotKube user in channel
func (b *MattermostBot) getChannel() *model.Channel {
	// Checking if channel exists
	botChannel, resp := b.APIClient.GetChannelByName(b.ChannelName, b.getTeam().Id, "")
	if resp.Error != nil {
		b.log.Fatalf("There was a problem finding Mattermost channel %s. %s", b.ChannelName, resp.Error)
	}

	// Adding BotKube user to channel
	b.APIClient.AddChannelMember(botChannel.Id, b.getUser().Id)
	return botChannel
}

func (b *MattermostBot) listen(ctx context.Context) {
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

// SendEvent sends event notification to MattermostBot
func (b *MattermostBot) SendEvent(ctx context.Context, event events.Event) error {
	if !b.notify {
		b.log.Info("Notifications are disabled. Skipping event...")
		return nil
	}

	b.log.Debugf(">> Sending to MattermostBot: %+v", event)

	var fields []*model.SlackAttachmentField

	switch b.Notification.Type {
	case config.LongNotification:
		fields = mmLongNotification(event)
	case config.ShortNotification:
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
		targetChannel = b.ChannelName
	}
	isDefaultChannel := targetChannel == b.ChannelName

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

// SendMessage sends message to MattermostBot channel
func (b *MattermostBot) SendMessage(_ context.Context, msg string) error {
	post := &model.Post{
		ChannelId: b.ChannelName,
		Message:   msg,
	}
	if _, resp := b.APIClient.CreatePost(post); resp.Error != nil {
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

	// Add clusterName in the message
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
