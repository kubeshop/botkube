package bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/execute"
	formatx "github.com/kubeshop/botkube/pkg/format"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
	"github.com/kubeshop/botkube/pkg/utils"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//    - handle and send methods from `slackMessage` should be defined on Bot level,
//    - split to multiple files in a separate package,
//    - review all the methods and see if they can be simplified.

var _ Bot = &Slack{}

const slackBotMentionPrefixFmt = "^<@%s>"

var attachmentColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "danger",
	config.Critical: "danger",
}

var (
	// SlackAppLevelToken for Slack Socket Mode. It is set during application build.
	SlackAppLevelToken string
)

// Slack listens for user's message, execute commands and sends back the response.
type Slack struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        FatalErrorAnalyticsReporter
	botID           string
	client          *slack.Client
	channelsMutex   sync.RWMutex
	channels        map[string]channelConfigByName
	notifyMutex     sync.Mutex
	botMentionRegex *regexp.Regexp
	commGroupName   string
	renderer        *SlackRenderer
}

type slackMessage struct {
	Text            string
	Channel         string
	ThreadTimeStamp string
}

// NewSlack creates a new Slack instance.
func NewSlack(log logrus.FieldLogger, commGroupName string, cfg config.Slack, executorFactory ExecutorFactory, reporter FatalErrorAnalyticsReporter) (*Slack, error) {
	botToken := cfg.Token
	appToken := SlackAppLevelToken
	if cfg.BotToken != "" && cfg.AppToken != "" {
		log.Info("Using custom bot and app tokens")
		botToken = cfg.BotToken
		appToken = cfg.AppToken
	}
	client := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	authResp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("while testing the ability to do auth Slack request: %w", err)
	}
	botID := authResp.UserID

	botMentionRegex, err := slackBotMentionRegex(botID)
	if err != nil {
		return nil, err
	}

	channels := slackChannelsConfigFrom(cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	return &Slack{
		log:             log,
		executorFactory: executorFactory,
		reporter:        reporter,
		botID:           botID,
		client:          client,
		channels:        channels,
		commGroupName:   commGroupName,
		renderer:        NewSlackRenderer(cfg.Notification),
		botMentionRegex: botMentionRegex,
	}, nil
}

// Start starts the Slack WebSocket connection and listens for messages
func (b *Slack) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	websocketClient := socketmode.New(b.client)

	go func() {
		defer analytics.ReportPanicIfOccurs(b.log, b.reporter)
		socketRunErr := websocketClient.RunContext(ctx)
		if socketRunErr != nil {
			reportErr := b.reporter.ReportFatalError(socketRunErr)
			if reportErr != nil {
				b.log.Errorf("while reporting socket error: %s", reportErr.Error())
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return nil
		case event := <-websocketClient.Events:
			switch event.Type {
			case socketmode.EventTypeConnecting:
				b.log.Info("BotKube is connecting to Slack...")
			case socketmode.EventTypeConnected:
				if err := b.reporter.ReportBotEnabled(b.IntegrationName()); err != nil {
					return fmt.Errorf("report analytics error: %w", err)
				}
				b.log.Info("BotKube connected to Slack!")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
				if !ok {
					b.log.Errorf("Invalid event %+v\n", event.Data)
					continue
				}
				websocketClient.Ack(*event.Request)
				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					b.log.Debugf("Got callback event %s", utils.StructDumper().Sdump(eventsAPIEvent))
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						b.log.Debugf("Got app mention %s", utils.StructDumper().Sdump(innerEvent))
						msg := slackMessage{
							Text:            ev.Text,
							Channel:         ev.Channel,
							ThreadTimeStamp: ev.ThreadTimeStamp,
						}
						if err := b.handleMessage(msg); err != nil {
							b.log.Errorf("Message handling error: %w", err)
						}
					}
				}
			case socketmode.EventTypeInteractive:
				callback, ok := event.Data.(slack.InteractionCallback)
				if !ok {
					b.log.Errorf("Invalid event %+v\n", event.Data)
					continue
				}

				websocketClient.Ack(*event.Request)

				switch callback.Type {
				case slack.InteractionTypeBlockActions:
					b.log.Debugf("Got block action %s", utils.StructDumper().Sdump(callback.ActionCallback.BlockActions))

					if len(callback.ActionCallback.BlockActions) != 1 {
						b.log.Debug("Ignoring callback as the number of actions is different from 1")
						continue
					}

					act := callback.ActionCallback.BlockActions[0]
					if strings.HasPrefix(act.ActionID, "url:") {
						continue // skip the url actions
					}
					msg := slackMessage{
						Text:            act.Value,
						Channel:         callback.Channel.ID,
						ThreadTimeStamp: callback.MessageTs,
					}
					if err := b.handleMessage(msg); err != nil {
						b.log.Errorf("Message handling error: %w", err)
					}
				default:
					b.log.Debugf("get unhandled event %s", callback.Type)
				}
			case socketmode.EventTypeErrorBadMessage:
				b.log.Errorf("Bad message: %+v\n", event.Data)
			case socketmode.EventTypeIncomingError:
				b.log.Errorf("Incoming error: %+v\n", event.Data)
			case socketmode.EventTypeConnectionError:
				b.log.Errorf("Slack connection error: %+v\n", event.Data)
			}
		}
	}
}

// Type describes the notifier type.
func (b *Slack) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// IntegrationName describes the notifier integration name.
func (b *Slack) IntegrationName() config.CommPlatformIntegration {
	return config.SlackCommPlatformIntegration
}

// NotificationsEnabled returns current notification status for a given channel name.
func (b *Slack) NotificationsEnabled(channelName string) bool {
	channel, exists := b.getChannels()[channelName]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel name.
func (b *Slack) SetNotificationsEnabled(channelName string, enabled bool) error {
	// avoid race conditions with using the setter concurrently, as we set whole map
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()

	channels := b.getChannels()
	channel, exists := channels[channelName]
	if !exists {
		return execute.ErrNotificationsNotConfigured
	}

	channel.notify = enabled
	channels[channelName] = channel
	b.setChannels(channels)

	return nil
}

func (b *Slack) handleMessage(event slackMessage) error {
	// Handle message only if starts with mention
	request, found := b.findAndTrimBotMention(event.Text)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}

	// Unfortunately we need to do a call for channel name based on ID every time a message arrives.
	// I wanted to query for channel IDs based on names and prepare a map in the `slackChannelsConfigFrom`,
	// but unfortunately BotKube would need another scope (get all conversations).
	// Keeping current way of doing this until we come up with a better idea.
	info, err := b.client.GetConversationInfo(event.Channel, true)
	if err != nil {
		return fmt.Errorf("while getting conversation info: %w", err)
	}

	channel, isAuthChannel := b.getChannels()[info.Name]

	e := b.executorFactory.NewDefault(b.commGroupName, b.IntegrationName(), b, isAuthChannel, info.Name, channel.Bindings.Executors, request)
	response := e.Execute()
	err = b.send(event, request, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *Slack) send(event slackMessage, req string, resp string) error {
	b.log.Debugf("Slack incoming Request: %s", req)
	b.log.Debugf("Slack Response: %s", resp)

	if resp == "" {
		return fmt.Errorf("while reading Slack response: empty response for request %q", req)
	}
	// Upload message as a file if too long
	if len(resp) >= 3990 {
		params := slack.FileUploadParameters{
			Filename: req,
			Title:    req,
			Content:  resp,
			Channels: []string{event.Channel},
		}
		_, err := b.client.UploadFile(params)
		if err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}
		return nil
	}

	options := []slack.MsgOption{
		slack.MsgOptionText(formatx.CodeBlock(resp), false),
	}

	//if the message is from thread then add an option to return the response to the thread
	if event.ThreadTimeStamp != "" {
		options = append(options, slack.MsgOptionTS(event.ThreadTimeStamp))
	}

	if _, _, err := b.client.PostMessage(event.Channel, options...); err != nil {
		return fmt.Errorf("while posting Slack message: %w", err)
	}

	return nil
}

// SendEvent sends event notification to slack
func (b *Slack) SendEvent(ctx context.Context, event events.Event, eventSources []string) error {
	b.log.Debugf("Sending to Slack: %+v", event)
	attachment := b.renderer.RenderEventMessage(event)

	errs := multierror.New()
	for _, channelName := range b.getChannelsToNotify(event, eventSources) {
		channelID, timestamp, err := b.client.PostMessageContext(ctx, channelName, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while posting message to channel %q: %w", channelName, err))
			continue
		}

		b.log.Debugf("Event successfully sent to channel %q (ID: %q) at %b", channelName, channelID, timestamp)
	}

	return errs.ErrorOrNil()
}

func (b *Slack) getChannelsToNotify(event events.Event, eventSources []string) []string {
	// support custom event routing
	if event.Channel != "" {
		return []string{event.Channel}
	}

	var out []string
	for _, cfg := range b.getChannels() {
		if !cfg.notify {
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
			continue
		}

		if !sliceutil.Intersect(eventSources, cfg.Bindings.Sources) {
			continue
		}

		out = append(out, cfg.Identifier())
	}
	return out
}

// SendMessage sends message to slack channel
func (b *Slack) SendMessage(ctx context.Context, msg string) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelName := channel.Name
		b.log.Debugf("Sending message to channel %q: %+v", channelName, msg)
		channelID, timestamp, err := b.client.PostMessageContext(ctx, channelName, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q: %w", channelName, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q at %q", channelID, timestamp)
	}

	return errs.ErrorOrNil()
}

// SendInteractiveMessage sends message with interactive sections to Slack channels.
func (b *Slack) SendInteractiveMessage(ctx context.Context, msg interactive.Message) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelName := channel.Name
		b.log.Debugf("Sending message to channel %q: %+v", channelName, msg)

		message := b.renderer.RenderInteractiveMessage(msg)

		channelID, timestamp, err := b.client.PostMessageContext(ctx, channelName, message)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q: %w", channelName, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q at %q", channelID, timestamp)
	}

	return errs.ErrorOrNil()
}

// BotName returns the Bot name.
func (b *Slack) BotName() string {
	return fmt.Sprintf("<@%s>", b.botID)
}

func (b *Slack) getChannels() map[string]channelConfigByName {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *Slack) setChannels(channels map[string]channelConfigByName) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}

func (b *Slack) findAndTrimBotMention(msg string) (string, bool) {
	if !b.botMentionRegex.MatchString(msg) {
		return "", false
	}

	return b.botMentionRegex.ReplaceAllString(msg, ""), true
}

func slackChannelsConfigFrom(channelsCfg config.IdentifiableMap[config.ChannelBindingsByName]) map[string]channelConfigByName {
	channels := make(map[string]channelConfigByName)
	for _, channCfg := range channelsCfg {
		channels[channCfg.Identifier()] = channelConfigByName{
			ChannelBindingsByName: channCfg,
			notify:                !channCfg.Notification.Disabled,
		}
	}

	return channels
}

func slackBotMentionRegex(botID string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(slackBotMentionPrefixFmt, botID))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}
