package bot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
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

var _ Bot = &Discord{}

const (
	// customTimeFormat holds custom time format string.
	customTimeFormat = "2006-01-02T15:04:05Z"

	// discordBotMentionRegexFmt supports also nicknames (the exclamation mark).
	// Read more: https://discordjs.guide/miscellaneous/parsing-mention-arguments.html#how-discord-mentions-work
	discordBotMentionRegexFmt = "^<@!?%s>"

	// discordMaxMessageSize max size before a message should be uploaded as a file.
	discordMaxMessageSize = 2000
)

var embedColor = map[config.Level]int{
	config.Info:     8311585,  // green
	config.Warn:     16312092, // yellow
	config.Debug:    8311585,  // green
	config.Error:    13632027, // red
	config.Critical: 13632027, // red
}

// Discord listens for user's message, execute commands and sends back the response.
type Discord struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	api             *discordgo.Session
	notification    config.Notification
	botID           string
	channelsMutex   sync.RWMutex
	channels        map[string]channelConfigByID
	notifyMutex     sync.Mutex
	botMentionRegex *regexp.Regexp
	commGroupName   string
	mdFormatter     interactive.MDFormatter
}

// discordMessage contains message details to execute command and send back the result.
type discordMessage struct {
	Event *discordgo.MessageCreate
}

// NewDiscord creates a new Discord instance.
func NewDiscord(log logrus.FieldLogger, commGroupName string, cfg config.Discord, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Discord, error) {
	botMentionRegex, err := discordBotMentionRegex(cfg.BotID)
	if err != nil {
		return nil, err
	}

	api, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}

	channelsCfg := discordChannelsConfigFrom(cfg.Channels)

	return &Discord{
		log:             log,
		reporter:        reporter,
		executorFactory: executorFactory,
		api:             api,
		botID:           cfg.BotID,
		notification:    cfg.Notification,
		commGroupName:   commGroupName,
		channels:        channelsCfg,
		botMentionRegex: botMentionRegex,
		mdFormatter:     interactive.DefaultMDFormatter(),
	}, nil
}

// Start starts the Discord websocket connection and listens for messages.
func (b *Discord) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	// Register the messageCreate func as a callback for MessageCreate events.
	b.api.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		msg := discordMessage{
			Event: m,
		}
		if err := b.handleMessage(ctx, msg); err != nil {
			b.log.Errorf("Message handling error: %s", err.Error())
		}
	})

	// Open a websocket connection to Discord and begin listening.
	err := b.api.Open()
	if err != nil {
		return fmt.Errorf("while opening connection: %w", err)
	}

	err = b.reporter.ReportBotEnabled(b.IntegrationName())
	if err != nil {
		return fmt.Errorf("while reporting analytics: %w", err)
	}

	b.log.Info("Botkube connected to Discord!")

	<-ctx.Done()
	b.log.Info("Shutdown requested. Finishing...")
	err = b.api.Close()
	if err != nil {
		return fmt.Errorf("while closing connection: %w", err)
	}

	return nil
}

// SendEvent sends event notification to Discord ChannelID.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendEvent(_ context.Context, event event.Event, eventSources []string) (err error) {
	b.log.Debugf("Sending to Discord: %+v", event)

	msgToSend := b.formatMessage(event)

	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotify(eventSources) {
		msg := msgToSend // copy as the struct is modified when using Discord API client
		if _, err := b.api.ChannelMessageSendComplex(channelID, &msg); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Discord message to channel %q: %w", channelID, err))
			continue
		}

		b.log.Debugf("Event successfully sent to channel %q", channelID)
	}

	return errs.ErrorOrNil()
}

// SendGenericMessage sends interactive message to selected Discord channels.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendGenericMessage(_ context.Context, genericMsg interactive.GenericMessage, sourceBindings []string) error {
	msg := genericMsg.ForBot(b.BotName())

	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotify(sourceBindings) {
		b.log.Debugf("Sending message to channel %q: %+v", channelID, msg)

		err := b.send(channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Discord message to channel %q: %w", channelID, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q", channelID)
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends interactive message to all Discord channels.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendMessageToAll(_ context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelID := channel.ID
		plaintext := interactive.RenderMessage(b.mdFormatter, msg)
		b.log.Debugf("Sending message to channel %q: %s", channelID, plaintext)

		if _, err := b.api.ChannelMessageSend(channelID, plaintext); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Discord message to channel %q: %w", channelID, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q", channelID)
	}

	return errs.ErrorOrNil()
}

// IntegrationName describes the integration name.
func (b *Discord) IntegrationName() config.CommPlatformIntegration {
	return config.DiscordCommPlatformIntegration
}

// Type describes the integration type.
func (b *Discord) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// TODO: Support custom routing via annotations for Discord as well
func (b *Discord) getChannelsToNotify(sourceBindings []string) []string {
	var out []string
	for _, cfg := range b.getChannels() {
		switch {
		case !cfg.notify:
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
		default:
			if sliceutil.Intersect(sourceBindings, cfg.Bindings.Sources) {
				out = append(out, cfg.Identifier())
			}
		}
	}
	return out
}

// NotificationsEnabled returns current notification status for a given channel ID.
func (b *Discord) NotificationsEnabled(channelID string) bool {
	channel, exists := b.getChannels()[channelID]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel ID.
func (b *Discord) SetNotificationsEnabled(channelID string, enabled bool) error {
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

// HandleMessage handles the incoming messages.
func (b *Discord) handleMessage(ctx context.Context, dm discordMessage) error {
	// Handle message only if starts with mention
	req, found := b.findAndTrimBotMention(dm.Event.Content)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}

	b.log.Debugf("Discord incoming Request: %s", req)

	channel, isAuthChannel := b.getChannels()[dm.Event.ChannelID]

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			ID:               channel.Identifier(),
			ExecutorBindings: channel.Bindings.Executors,
			SourceBindings:   channel.Bindings.Sources,
			IsAuthenticated:  isAuthChannel,
			CommandOrigin:    command.TypedOrigin,
		},
		Message: req,
		User:    fmt.Sprintf("<@%s>", dm.Event.Author.ID),
	})

	response := e.Execute(ctx)
	err := b.send(dm.Event.ChannelID, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *Discord) send(channelID string, resp interactive.CoreMessage) error {
	b.log.Debugf("Discord Response: %s", resp)

	markdown := interactive.RenderMessage(b.mdFormatter, resp)

	if len(markdown) == 0 {
		return errors.New("while reading Discord response: empty response")
	}

	// Upload message as a file if too long
	if len(markdown) >= discordMaxMessageSize {
		params := &discordgo.MessageSend{
			Content: resp.Description,
			Files: []*discordgo.File{
				{
					Name:   "Response.txt",
					Reader: strings.NewReader(interactive.MessageToPlaintext(resp, interactive.NewlineFormatter)),
				},
			},
		}
		if _, err := b.api.ChannelMessageSendComplex(channelID, params); err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}
		return nil
	}

	if _, err := b.api.ChannelMessageSend(channelID, markdown); err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}
	return nil
}

// BotName returns the Bot name.
func (b *Discord) BotName() string {
	// Note: we can use the botID, but it's not rendered well.
	// We would need to execute external call to find the Bot display name.
	// But this will be solved once we will introduce full support
	// for interactive messages.
	return "@Botkube"
}

func (b *Discord) getChannels() map[string]channelConfigByID {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *Discord) setChannels(channels map[string]channelConfigByID) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}

func (b *Discord) findAndTrimBotMention(msg string) (string, bool) {
	if !b.botMentionRegex.MatchString(msg) {
		return "", false
	}

	return b.botMentionRegex.ReplaceAllString(msg, ""), true
}

func discordChannelsConfigFrom(channelsCfg config.IdentifiableMap[config.ChannelBindingsByID]) map[string]channelConfigByID {
	res := make(map[string]channelConfigByID)
	for channAlias, channCfg := range channelsCfg {
		res[channCfg.Identifier()] = channelConfigByID{
			ChannelBindingsByID: channCfg,
			alias:               channAlias,
			notify:              !channCfg.Notification.Disabled,
		}
	}

	return res
}

func discordBotMentionRegex(botID string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(discordBotMentionRegexFmt, botID))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}
