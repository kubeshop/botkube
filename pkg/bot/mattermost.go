package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"

	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
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
	responseFileName             = "response.txt"
)

// TODO:
// 	- Use latest Mattermost API v6
// 	- Remove usage of `log.Fatal` - return error instead

// Mattermost listens for user's message, execute commands and sends back the response.
type Mattermost struct {
	log               logrus.FieldLogger
	executorFactory   ExecutorFactory
	reporter          AnalyticsReporter
	serverURL         string
	botName           string
	botUserID         string
	teamName          string
	webSocketURL      string
	wsClient          *model.WebSocketClient
	apiClient         *model.Client4
	channelsMutex     sync.RWMutex
	commGroupMetadata CommGroupMetadata
	channels          map[string]channelConfigByID
	notifyMutex       sync.Mutex
	botMentionRegex   *regexp.Regexp
	renderer          *MattermostRenderer
	userNamesForID    map[string]string
	messages          chan mattermostMessage
	messageWorkers    *pool.Pool
	shutdownOnce      sync.Once
	status            health.PlatformStatusMsg
	failureReason     health.FailureReasonMsg
}

// mattermostMessage contains message details to execute command and send back the result
type mattermostMessage struct {
	Event *model.WebSocketEvent
}

// NewMattermost creates a new Mattermost instance.
func NewMattermost(ctx context.Context, log logrus.FieldLogger, commGroupMetadata CommGroupMetadata, cfg config.Mattermost, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Mattermost, error) {
	botMentionRegex, err := mattermostBotMentionRegex(cfg.BotName)
	if err != nil {
		return nil, err
	}

	mmURL := strings.TrimRight(cfg.URL, "/") // This is already done in `model.NewWebSocketClient4`, but we also need it for WebSocket connection
	checkURL, err := url.Parse(mmURL)
	if err != nil {
		return nil, fmt.Errorf("while parsing Mattermost URL %q: %w", mmURL, err)
	}

	// Create WebSocketClient and handle messages
	webSocketURL := WebSocketProtocol + checkURL.Host + checkURL.Path
	if checkURL.Scheme == httpsScheme {
		webSocketURL = WebSocketSecureProtocol + checkURL.Host + checkURL.Path
	}

	log.WithFields(logrus.Fields{
		"webSocketURL": webSocketURL,
		"apiURL":       mmURL,
	}).Debugf("Setting up Mattermost bot...")

	client := model.NewAPIv4Client(mmURL)
	client.SetOAuthToken(cfg.Token)

	// In Mattermost v7.0+, what we see in MM Console is `display_name` of a team.
	// We need `name` of the team to make the rest of the business logic work.
	team, err := getMattermostTeam(ctx, client, cfg.Team)
	if err != nil {
		return nil, fmt.Errorf("while getting team details: %w", err)
	}

	channelsByIDCfg, err := mattermostChannelsCfgFrom(ctx, client, team.Id, cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	botUserID, err := getBotUserID(ctx, client, team.Id, cfg.BotName)
	if err != nil {
		return nil, fmt.Errorf("while getting bot user ID: %w", err)
	}

	return &Mattermost{
		log:               log,
		executorFactory:   executorFactory,
		reporter:          reporter,
		serverURL:         cfg.URL,
		botName:           cfg.BotName,
		botUserID:         botUserID,
		teamName:          team.Name,
		apiClient:         client,
		webSocketURL:      webSocketURL,
		commGroupMetadata: commGroupMetadata,
		channels:          channelsByIDCfg,
		botMentionRegex:   botMentionRegex,
		renderer:          NewMattermostRenderer(),
		userNamesForID:    map[string]string{},
		messages:          make(chan mattermostMessage, platformMessageChannelSize),
		messageWorkers:    pool.New().WithMaxGoroutines(platformMessageWorkersCount),
		status:            health.StatusUnknown,
		failureReason:     "",
	}, nil
}

func (b *Mattermost) startMessageProcessor(ctx context.Context) {
	b.log.Info("Starting mattermost message processor...")
	defer b.log.Info("Stopped mattermost message processor...")

	for msg := range b.messages {
		b.messageWorkers.Go(func() {
			err := b.handleMessage(ctx, msg)
			if err != nil {
				b.log.WithError(err).Error("Failed to handle Mattermost message")
			}
		})
	}
} // Start establishes mattermost connection and listens for messages
func (b *Mattermost) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	// Check connection to Mattermost server
	err := b.checkServerConnection(ctx)
	if err != nil {
		b.setStatusReason(health.FailureReasonConnectionError)
		return fmt.Errorf("while pinging Mattermost server %q: %w", b.serverURL, err)
	}

	err = b.reporter.ReportBotEnabled(b.IntegrationName(), b.commGroupMetadata.Index)
	if err != nil {
		b.log.Errorf("report analytics error: %s", err.Error())
	}

	// It is observed that Mattermost server closes connections unexpectedly after some time.
	// For now, we are adding retry logic to reconnect to the server
	// https://github.com/kubeshop/botkube/issues/201
	b.log.Info("Botkube connected to Mattermost!")
	b.setStatusReason("")
	go b.startMessageProcessor(ctx)

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			b.shutdown()
			return nil
		default:
			var appErr error
			b.wsClient, appErr = model.NewWebSocketClient4(b.webSocketURL, b.apiClient.AuthToken)
			if appErr != nil {
				b.setStatusReason(health.FailureReasonConnectionError)
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
func (b *Mattermost) handleMessage(ctx context.Context, mm mattermostMessage) error {
	post, err := postFromEvent(mm.Event)
	if err != nil {
		return fmt.Errorf("while getting post from event: %w", err)
	}

	// Skip if message posted by Botkube
	if post.UserId == b.botUserID {
		return nil
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
	if !exists {
		channel = channelConfigByID{
			ChannelBindingsByID: config.ChannelBindingsByID{
				ID: channelID,
			},
		}
	}

	userName, err := b.getUserName(ctx, post.UserId)
	if err != nil {
		b.log.Errorf("while getting user name: %s", err.Error())
	}
	if userName == "" {
		userName = post.UserId
	}

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupMetadata.Name,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			DisplayName:      channel.name,
			ID:               channel.Identifier(),
			ExecutorBindings: channel.Bindings.Executors,
			SourceBindings:   channel.Bindings.Sources,
			IsKnown:          exists,
			CommandOrigin:    command.TypedOrigin,
		},
		User: execute.UserInput{
			//Mention:     "", // not used currently
			DisplayName: userName,
		},
		Message: req,
	})
	response := e.Execute(ctx)
	err = b.send(ctx, channelID, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

// Send messages to Mattermost
func (b *Mattermost) send(ctx context.Context, channelID string, resp interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", channelID, resp)

	resp.ReplaceBotNamePlaceholder(b.BotName())
	post, err := b.formatMessage(ctx, resp, channelID)
	if err != nil {
		return fmt.Errorf("while formatting message: %w", err)
	}

	if _, _, err := b.apiClient.CreatePost(ctx, post); err != nil {
		b.log.Error("Failed to send message. Error: ", err)
	}

	b.log.Debugf("Message successfully sent to channel %q", channelID)
	return nil
}

func (b *Mattermost) formatMessage(ctx context.Context, msg interactive.CoreMessage, channelID string) (*model.Post, error) {
	// 1. Check the size and upload message as a file if it's too long
	plaintext := interactive.MessageToPlaintext(msg, interactive.NewlineFormatter)
	if len(plaintext) == 0 {
		return nil, errors.New("while reading Mattermost response: empty response")
	}
	if len(plaintext) >= mattermostMaxMessageSize {
		uploadResponse, _, err := b.apiClient.UploadFileAsRequestBody(
			ctx,
			[]byte(plaintext),
			channelID,
			responseFileName,
		)
		if err != nil {
			return nil, fmt.Errorf("while uploading file: %w", err)
		}

		return &model.Post{
			ChannelId: channelID,
			Message:   msg.Description,
			FileIds:   []string{uploadResponse.FileInfos[0].Id},
		}, nil
	}

	// 2. If it's not a simplified event, render as markdown
	if msg.Type != api.NonInteractiveSingleSection {
		return &model.Post{
			ChannelId: channelID,
			Message:   b.renderer.MessageToMarkdown(msg),
		}, nil
	}

	// FIXME: For now, we just render only with a few fields that are always present in the event message.
	// This should be removed once we will add support for rendering AdaptiveCard with all message primitives.
	attachments, err := b.renderer.NonInteractiveSectionToCard(msg)
	if err != nil {
		return nil, fmt.Errorf("while rendering event message embed: %w", err)
	}

	return &model.Post{
		Props: map[string]interface{}{
			"attachments": attachments,
		},
		ChannelId: channelID,
	}, nil
}

// Check if Mattermost server is reachable
func (b *Mattermost) checkServerConnection(ctx context.Context) error {
	// Check api connection
	if _, _, err := b.apiClient.GetOldClientConfig(ctx, ""); err != nil {
		return err
	}

	// Get channel list
	_, _, err := b.apiClient.GetTeamByName(ctx, b.teamName, "")
	if err != nil {
		return err
	}
	return nil
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

			mm := mattermostMessage{
				Event: event,
			}
			b.messages <- mm
		}
	}
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
func (b *Mattermost) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotify(sourceBindings) {
		err := b.send(ctx, channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Mattermost message to channel %q: %w", channelID, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends message to all Mattermost channels.
func (b *Mattermost) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelID := channel.ID
		err := b.send(ctx, channelID, msg)
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

func (b *Mattermost) getUserName(ctx context.Context, userID string) (string, error) {
	userName, exists := b.userNamesForID[userID]
	if exists {
		return userName, nil
	}

	user, _, err := b.apiClient.GetUser(ctx, userID, "")
	if err != nil {
		return "", fmt.Errorf("while getting user with ID %q: %w", userID, err)
	}
	b.userNamesForID[userID] = user.Username

	return user.Username, nil
}

func (b *Mattermost) shutdown() {
	b.shutdownOnce.Do(func() {
		b.log.Info("Shutting down mattermost message processor...")
		close(b.messages)
		b.messageWorkers.Wait()
	})
}

func getBotUserID(ctx context.Context, client *model.Client4, teamID, botName string) (string, error) {
	users, _, err := client.GetUsersByUsernames(ctx, []string{botName})
	if err != nil {
		return "", fmt.Errorf("while getting user with name %q: %w", botName, err)
	}

	if len(users) == 0 {
		return "", fmt.Errorf("user with name %q not found", botName)
	}

	teamMember, _, err := client.GetTeamMember(ctx, teamID, users[0].Id, "")
	if err != nil {
		return "", fmt.Errorf("while validating user with name %q is in team %q: %w", botName, teamID, err)
	}

	return teamMember.UserId, nil
}

func getMattermostTeam(ctx context.Context, client *model.Client4, name string) (*model.Team, error) {
	botTeams, r, err := client.SearchTeams(ctx, &model.TeamSearch{
		Term: name, // the search term to match against the name or display name of teams
	})
	if err != nil {
		return nil, fmt.Errorf("while searching team by term: %w", err)
	}

	if r.StatusCode == http.StatusNotImplemented || len(botTeams) == 0 {
		// try to check if we can get a given team directly
		botTeam, _, err := client.GetTeamByName(ctx, name, "")
		if err != nil {
			return nil, fmt.Errorf("while getting team by name: %w", err)
		}
		botTeams = append(botTeams, botTeam)
	}
	if len(botTeams) == 0 {
		return nil, fmt.Errorf("team %q not found", name)
	}

	return botTeams[0], err
}

func mattermostChannelsCfgFrom(ctx context.Context, client *model.Client4, teamID string, channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) (map[string]channelConfigByID, error) {
	res := make(map[string]channelConfigByID)
	for channAlias, channCfg := range channelsCfg {
		// do not normalize channel as Mattermost allows virtually all characters in channel names
		// See https://docs.mattermost.com/channels/channel-naming-conventions.html
		fetchedChannel, _, err := client.GetChannelByName(ctx, channCfg.Identifier(), teamID, "")
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
			name:   channCfg.Identifier(),
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

func (b *Mattermost) setStatusReason(reason health.FailureReasonMsg) {
	if reason == "" {
		b.status = health.StatusHealthy
	} else {
		b.status = health.StatusUnHealthy
	}
	b.failureReason = reason
}

// GetStatus gets bot status.
func (b *Mattermost) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   b.status,
		Restarts: "0/0",
		Reason:   b.failureReason,
	}
}
