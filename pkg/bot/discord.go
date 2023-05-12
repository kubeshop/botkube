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

var _ Bot = &Discord{}

const (
	// discordBotMentionRegexFmt supports also nicknames (the exclamation mark).
	// Read more: https://discordjs.guide/miscellaneous/parsing-mention-arguments.html#how-discord-mentions-work
	discordBotMentionRegexFmt = "^<@!?%s>"

	// discordMaxMessageSize max size before a message should be uploaded as a file.
	discordMaxMessageSize = 2000
)

// Discord listens for user's message, execute commands and sends back the response.
type Discord struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	api             *discordgo.Session
	botID           string
	channelsMutex   sync.RWMutex
	channels        map[string]channelConfigByID
	notifyMutex     sync.Mutex
	botMentionRegex *regexp.Regexp
	commGroupName   string
	renderer        *DiscordRenderer
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

	channelsCfg, err := discordChannelsConfigFrom(api, cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord channels config: %w", err)
	}

	return &Discord{
		log:             log,
		reporter:        reporter,
		executorFactory: executorFactory,
		api:             api,
		botID:           cfg.BotID,
		commGroupName:   commGroupName,
		channels:        channelsCfg,
		botMentionRegex: botMentionRegex,
		renderer:        NewDiscordRenderer(),
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

// SendMessage sends interactive message to selected Discord channels.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendMessage(_ context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	errs := multierror.New()
	for _, channelID := range b.getChannelsToNotify(sourceBindings) {
		err := b.send(channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Discord message to channel %q: %w", channelID, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends interactive message to all Discord channels.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendMessageToAll(_ context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelID := channel.ID

		err := b.send(channelID, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Discord message to channel %q: %w", channelID, err))
			continue
		}
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

	channel, exists := b.getChannels()[dm.Event.ChannelID]
	if !exists {
		channel = channelConfigByID{
			ChannelBindingsByID: config.ChannelBindingsByID{
				ID: dm.Event.ChannelID,
			},
		}
	}

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
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
		Message: req,
		User: execute.UserInput{
			Mention:     fmt.Sprintf("<@%s>", dm.Event.Author.ID),
			DisplayName: dm.Event.Author.String(),
		},
	})

	response := e.Execute(ctx)
	err := b.send(dm.Event.ChannelID, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *Discord) send(channelID string, resp interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", channelID, resp)

	resp.ReplaceBotNamePlaceholder(b.BotName())

	discordMsg, err := b.formatMessage(resp)
	if err != nil {
		return fmt.Errorf("while formatting message: %w", err)
	}
	if _, err := b.api.ChannelMessageSendComplex(channelID, discordMsg); err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	b.log.Debugf("Message successfully sent to channel %q", channelID)
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

func (b *Discord) formatMessage(msg interactive.CoreMessage) (*discordgo.MessageSend, error) {
	// 1. Check the size and upload message as a file if it's too long
	plaintext := interactive.MessageToPlaintext(msg, interactive.NewlineFormatter)
	if len(plaintext) == 0 {
		return nil, errors.New("while reading Discord response: empty response")
	}
	if len(plaintext) >= discordMaxMessageSize {
		return &discordgo.MessageSend{
			Content: msg.Description,
			Files: []*discordgo.File{
				{
					Name:   "Response.txt",
					Reader: strings.NewReader(plaintext),
				},
			},
		}, nil
	}

	// 2. If it's not a simplified event, render as markdown
	if msg.Type != api.NonInteractiveSingleSection {
		return &discordgo.MessageSend{
			Content: b.renderer.MessageToMarkdown(msg),
		}, nil
	}

	// FIXME: For now, we just render only with a few fields that are always present in the event message.
	// This should be removed once we will add support for rendering AdaptiveCard with all message primitives.
	messageEmbed, err := b.renderer.NonInteractiveSectionToCard(msg)
	if err != nil {
		return nil, fmt.Errorf("while rendering event message embed: %w", err)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			&messageEmbed,
		},
	}, nil
}

func discordChannelsConfigFrom(api *discordgo.Session, channelsCfg config.IdentifiableMap[config.ChannelBindingsByID]) (map[string]channelConfigByID, error) {
	res := make(map[string]channelConfigByID)
	for channAlias, channCfg := range channelsCfg {
		channelData, err := api.Channel(channCfg.Identifier())
		if err != nil {
			return nil, fmt.Errorf("while getting channel name for ID %q: %w", channCfg.Identifier(), err)
		}

		res[channCfg.Identifier()] = channelConfigByID{
			ChannelBindingsByID: channCfg,
			alias:               channAlias,
			notify:              !channCfg.Notification.Disabled,
			name:                channelData.Name,
		}
	}

	return res, nil
}

func discordBotMentionRegex(botID string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(discordBotMentionRegexFmt, botID))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}
