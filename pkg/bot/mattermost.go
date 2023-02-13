package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//    - split to multiple files in a separate package,
//    - review all the methods and see if they can be simplified.

var _ Bot = &Mattermost{}

const (
	// WebSocketProtocol stores protocol initials for web socket
	WebSocketProtocol = "ws://"
	// WebSocketSecureProtocol stores protocol initials for web socket
	WebSocketSecureProtocol = "wss://"
	// mattermostMaxMessageSize max size before a message should be uploaded as a file.
	mattermostMaxMessageSize = 3990

	httpsScheme                  = "https"
	mattermostBotMentionRegexFmt = "^@(?i)%s"
)

// TODO:
// 	- Use latest Mattermost API v6
// 	- Remove usage of `log.Fatal` - return error instead

// Mattermost listens for user's message, execute commands and sends back the response.
type Mattermost struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	notification    config.Notification
	serverURL       string
	botName         string
	teamName        string
	webSocketURL    string
	wsClient        *model.WebSocketClient
	apiClient       *model.Client4
	channelsMutex   sync.RWMutex
	commGroupName   string
	channels        map[string]channelConfigByID
	notifyMutex     sync.Mutex
	botMentionRegex *regexp.Regexp
	mdFormatter     interactive.MDFormatter
}

// mattermostMessage contains message details to execute command and send back the result
type mattermostMessage struct {
	Event         *model.WebSocketEvent
	IsAuthChannel bool
}

// NewMattermost creates a new Mattermost instance.
func NewMattermost(log logrus.FieldLogger, commGroupName string, cfg config.Mattermost, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Mattermost, error) {
	botMentionRegex, err := mattermostBotMentionRegex(cfg.BotName)
	if err != nil {
		return nil, err
	}

	checkURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("while parsing Mattermost URL %q: %w", cfg.URL, err)
	}

	// Create WebSocketClient and handle messages
	webSocketURL := WebSocketProtocol + checkURL.Host
	if checkURL.Scheme == httpsScheme {
		webSocketURL = WebSocketSecureProtocol + checkURL.Host
	}

	client := model.NewAPIv4Client(cfg.URL)
	client.SetOAuthToken(cfg.Token)

	botTeams, _, err := client.SearchTeams(&model.TeamSearch{
		Term: cfg.Team,
	})
	if err != nil {
		return nil, fmt.Errorf("while getting team by name: %w", err)
	}

	if len(botTeams) == 0 {
		return nil, fmt.Errorf("team: %s not found", cfg.Team)
	}
	botTeam := botTeams[0]
	// In Mattermost v7.0+, what we see in MM Console is `display_name` of team.
	// We need `name` of team to make rest of the business logic work.
	cfg.Team = botTeam.Name
	channelsByIDCfg, err := mattermostChannelsCfgFrom(client, botTeam.Id, cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	return &Mattermost{
		log:             log,
		executorFactory: executorFactory,
		reporter:        reporter,
		notification:    cfg.Notification,
		serverURL:       cfg.URL,
		botName:         cfg.BotName,
		teamName:        cfg.Team,
		apiClient:       client,
		webSocketURL:    webSocketURL,
		commGroupName:   commGroupName,
		channels:        channelsByIDCfg,
		botMentionRegex: botMentionRegex,
		mdFormatter:     interactive.DefaultMDFormatter(),
	}, nil
}

// Start establishes mattermost connection and listens for messages
func (b *Mattermost) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	// Check connection to Mattermost server
	err := b.checkServerConnection()
	if err != nil {
		return fmt.Errorf("while pinging Mattermost server %q: %w", b.serverURL, err)
	}

	err = b.reporter.ReportBotEnabled(b.IntegrationName())
	if err != nil {
		return fmt.Errorf("while reporting analytics: %w", err)
	}

	// It is observed that Mattermost server closes connections unexpectedly after some time.
	// For now, we are adding retry logic to reconnect to the server
	// https://github.com/kubeshop/botkube/issues/201
	b.log.Info("Botkube connected to Mattermost!")
	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return nil
		default:
			var appErr error
			b.wsClient, appErr = model.NewWebSocketClient4(b.webSocketURL, b.apiClient.AuthToken)
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

// NotificationsEnabled returns current notification status for a given channel ID.
func (b *Mattermost) NotificationsEnabled(channelID string) bool {
	channel, exists := b.getChannels()[channelID]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel ID.
func (b *Mattermost) SetNotificationsEnabled(channelID string, enabled bool) error {
	// avoid race conditions with using the setter concurrently, as we set whole map
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()

	channels := b.getChannels()
	channel, exists := channels[channelID]
	if !exists {
		return execute.ErrNotificationsNotConfigured
	}

	channel.notify = enabled
	channels[channelID] = channel
	b.setChannels(channels)

	return nil
}

// Check incoming message and take action
func (b *Mattermost) handleMessage(ctx context.Context, mm *mattermostMessage) error {
	post, err := postFromEvent(mm.Event)
	if err != nil {
		return fmt.Errorf("while getting post from event: %w", err)
	}

	// Handle message only if starts with mention
	trimmedMsg, found := b.findAndTrimBotMention(post.Message)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botName)
		return nil
	}
	req := trimmedMsg
	b.log.Debugf("Mattermost incoming Request: %s", req)

	channelID := mm.Event.GetBroadcast().ChannelId
	channel, exists := b.getChannels()[channelID]
	mm.IsAuthChannel = exists

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			ID:               channel.Identifier(),
			ExecutorBindings: channel.Bindings.Executors,
			SourceBindings:   channel.Bindings.Sources,
			IsAuthenticated:  mm.IsAuthChannel,
			CommandOrigin:    command.TypedOrigin,
		},
		Message: req,
	})
	response := e.Execute(ctx)
	err = b.send(channelID, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

// Send messages to Mattermost
func (b *Mattermost) send(channelID string, resp interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", channelID, resp)

	resp.ReplaceBotNamePlaceholder(b.BotName())
	markdown := interactive.RenderMessage(b.mdFormatter, resp)

	if len(markdown) == 0 {
		return errors.New("while reading Mattermost response: empty response")
	}

	// Create file if message is too large
	if len(markdown) >= mattermostMaxMessageSize {
		uploadResponse, _, err := b.apiClient.UploadFileAsRequestBody(
			[]byte(interactive.MessageToPlaintext(resp, interactive.NewlineFormatter)),
			channelID,
			responseFileName,
		)
		if err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}

		post := &model.Post{}
		post.ChannelId = channelID
		post.Message = resp.Description
		post.FileIds = []string{uploadResponse.FileInfos[0].Id}

		if _, _, err := b.apiClient.CreatePost(post); err != nil {
			return fmt.Errorf("while sending attachment message: %w", err)
		}

		return nil
	}

	post := &model.Post{
		ChannelId: channelID,
		Message:   markdown,
	}
	if _, _, err := b.apiClient.CreatePost(post); err != nil {
		b.log.Error("Failed to send message. Error: ", err)
	}
	b.log.Debugf("Message successfully sent to channel %q", channelID)
	return nil
}

// Check if Mattermost server is reachable
func (b *Mattermost) checkServerConnection() error {
	// Check api connection
	if _, _, err := b.apiClient.GetOldClientConfig(""); err != nil {
		return err
	}

	// Get channel list
	_, _, err := b.apiClient.GetTeamByName(b.teamName, "")
	if err != nil {
		return err
	}
	return nil
}

// Check if team exists in Mattermost
func (b *Mattermost) getTeam() *model.Team {
	botTeam, _, err := b.apiClient.GetTeamByName(b.teamName, "")
	if err != nil {
		b.log.Fatalf("There was a problem finding Mattermost team %s. %s", b.teamName, err)
	}
	return botTeam
}

// Check if Botkube user exists in Mattermost
func (b *Mattermost) getUser() *model.User {
	users, _, err := b.apiClient.AutocompleteUsersInTeam(b.getTeam().Id, b.botName, 1, "")
	if err != nil {
		b.log.Fatalf("There was a problem finding Mattermost user %s. %s", b.botName, err)
	}
	return users.Users[0]
}

func (b *Mattermost) listen(ctx context.Context) {
	b.wsClient.Listen()
	defer b.wsClient.Close()
	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return
		case event, ok := <-b.wsClient.EventChannel:
			if !ok {
				if b.wsClient.ListenError != nil {
					b.log.Debugf("while listening on websocket connection: %s", b.wsClient.ListenError.Error())
				}

				b.log.Info("Incoming events channel closed. Finishing...")
				return
			}

			if event == nil {
				b.log.Info("Nil event, ignoring")
				continue
			}

			if event.EventType() != model.WebsocketEventPosted {
				// ignore
				continue
			}

			post, err := postFromEvent(event)
			if err != nil {
				continue
			}

			// Skip if message posted by Botkube or doesn't start with mention
			if post.UserId == b.getUser().Id {
				continue
			}
			mm := &mattermostMessage{
				Event:         event,
				IsAuthChannel: false,
			}
			err = b.handleMessage(ctx, mm)
			if err != nil {
				wrappedErr := fmt.Errorf("while handling message: %w", err)
				b.log.Errorf(wrappedErr.Error())
			}
		}
	}
}

// SendEvent sends event notification to Mattermost
func (b *Mattermost) SendEvent(_ context.Context, event event.Event, eventSources []string) error {
	b.log.Debugf("Sending to Mattermost: %+v", event)
	attachment := b.formatAttachments(event)

	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotifyForEvent(event, eventSources) {
		post := &model.Post{
			Props: map[string]interface{}{
				"attachments": attachment,
			},
			ChannelId: channelID,
		}

		_, _, err := b.apiClient.CreatePost(post)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while posting message to channel %q: %w", channelID, err))
			continue
		}

		b.log.Debugf("Event successfully sent to channel %q", post.ChannelId)
	}

	return errs.ErrorOrNil()
}

func (b *Mattermost) getChannelsToNotifyForEvent(event event.Event, sourceBindings []string) []string {
	// support custom event routing
	if event.Channel != "" {
		return []string{event.Channel}
	}

	return b.getChannelsToNotify(sourceBindings)
}

func (b *Mattermost) getChannelsToNotify(eventSources []string) []string {
	var out []string
	for _, cfg := range b.getChannels() {
		switch {
		case !cfg.notify:
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
		default:
			if sliceutil.Intersect(eventSources, cfg.Bindings.Sources) {
				out = append(out, cfg.Identifier())
			}
		}
	}
	return out
}

// SendMessage sends message to selected Mattermost channels.
func (b *Mattermost) SendMessage(_ context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotify(sourceBindings) {
		err := b.send(channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Mattermost message to channel %q: %w", channelID, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends message to all Mattermost channels.
func (b *Mattermost) SendMessageToAll(_ context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelID := channel.ID
		err := b.send(channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Mattermost message to channel %q: %w", channelID, err))
			continue
		}
	}
	return errs.ErrorOrNil()
}

// BotName returns the Bot name.
func (b *Mattermost) BotName() string {
	return fmt.Sprintf("@%s", b.botName)
}

func (b *Mattermost) findAndTrimBotMention(msg string) (string, bool) {
	if !b.botMentionRegex.MatchString(msg) {
		return "", false
	}

	return b.botMentionRegex.ReplaceAllString(msg, ""), true
}

func (b *Mattermost) getChannels() map[string]channelConfigByID {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *Mattermost) setChannels(channels map[string]channelConfigByID) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}

func mattermostChannelsCfgFrom(client *model.Client4, teamID string, channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) (map[string]channelConfigByID, error) {
	res := make(map[string]channelConfigByID)
	for channAlias, channCfg := range channelsCfg {
		fetchedChannel, _, err := client.GetChannelByName(channCfg.Identifier(), teamID, "")
		if err != nil {
			return nil, fmt.Errorf("while getting channel by name %q: %w", channCfg.Name, err)
		}

		res[fetchedChannel.Id] = channelConfigByID{
			ChannelBindingsByID: config.ChannelBindingsByID{
				ID:       fetchedChannel.Id,
				Bindings: channCfg.Bindings,
			},
			alias:  channAlias,
			notify: !channCfg.Notification.Disabled,
		}
	}

	return res, nil
}

func mattermostBotMentionRegex(botName string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(mattermostBotMentionRegexFmt, botName))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}

func postFromEvent(event *model.WebSocketEvent) (*model.Post, error) {
	var post *model.Post
	if err := json.NewDecoder(strings.NewReader(event.GetData()["post"].(string))).Decode(&post); err != nil {
		return nil, fmt.Errorf("while getting post from event: %w", err)
	}
	return post, nil
}
