package bot

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/execute"
	formatx "github.com/kubeshop/botkube/pkg/format"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
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

// Slack listens for user's message, execute commands and sends back the response.
type Slack struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        FatalErrorAnalyticsReporter
	botID           string
	client          *slack.Client
	notification    config.Notification
	channelsMutex   sync.RWMutex
	channels        map[string]channelConfigByName
	notifyMutex     sync.Mutex
	botMentionRegex *regexp.Regexp
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	Event    *slackevents.AppMentionEvent
	BotID    string
	Request  string
	Response string
	client   *slack.Client
}

// NewSlack creates a new Slack instance.
func NewSlack(log logrus.FieldLogger, cfg config.Slack, executorFactory ExecutorFactory, reporter FatalErrorAnalyticsReporter) (*Slack, error) {
	client := slack.New(cfg.BotToken, slack.OptionAppLevelToken(cfg.AppToken))

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
		notification:    cfg.Notification,
		channels:        channels,
		botMentionRegex: botMentionRegex,
	}, nil
}

// Start starts the Slack Websocket connection and listens for messages
func (b *Slack) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	websocketClient := socketmode.New(b.client)

	go func() {
		defer analytics.ReportPanicIfOccurs(b.log, b.reporter)
		socketRunErr := websocketClient.Run()
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
					b.log.Errorf("Invalid event %T", event.Data)
					continue
				}
				websocketClient.Ack(*event.Request)
				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					b.log.Debugf("Got callback event %w", eventsAPIEvent)
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						b.log.Debugf("Got app mention %w", innerEvent)
						sm := slackMessage{
							log:             b.log,
							executorFactory: b.executorFactory,
							Event:           ev,
							BotID:           b.botID,
							client:          b.client,
						}
						if err := sm.HandleMessage(b); err != nil {
							b.log.Errorf("Message handling error: %w", err)
						}
					}
				}
			case socketmode.EventTypeErrorBadMessage:
				b.log.Errorf("Bad message: %w", event.Data)
			case socketmode.EventTypeIncomingError:
				b.log.Errorf("Incoming error: %w", event.Data)
			case socketmode.EventTypeConnectionError:
				b.log.Errorf("Slack connection error: %w", event.Data)
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

func (sm *slackMessage) HandleMessage(b *Slack) error {
	// Handle message only if starts with mention
	trimmedMsg, found := b.findAndTrimBotMention(sm.Event.Text)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}
	sm.Request = trimmedMsg

	// Unfortunately we need to do a call for channel name based on ID every time a message arrives.
	// I wanted to query for channel IDs based on names and prepare a map in the `slackChannelsConfigFrom`,
	// but unfortunately BotKube would need another scope (get all conversations).
	// Keeping current way of doing this until we come up with a better idea.
	channelID := sm.Event.Channel
	info, err := b.client.GetConversationInfo(channelID, true)
	if err != nil {
		return fmt.Errorf("while getting conversation info: %w", err)
	}

	channel, isAuthChannel := b.getChannels()[info.Name]

	e := sm.executorFactory.NewDefault(b.IntegrationName(), b, isAuthChannel, info.Name, channel.Bindings.Executors, sm.Request)
	sm.Response = e.Execute()
	err = sm.Send()
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (sm *slackMessage) Send() error {
	sm.log.Debugf("Slack incoming Request: %s", sm.Request)
	sm.log.Debugf("Slack Response: %s", sm.Response)
	if len(sm.Response) == 0 {
		return fmt.Errorf("while reading Slack response: empty response for request %q", sm.Request)
	}
	// Upload message as a file if too long
	if len(sm.Response) >= 3990 {
		params := slack.FileUploadParameters{
			Filename: sm.Request,
			Title:    sm.Request,
			Content:  sm.Response,
			Channels: []string{sm.Event.Channel},
		}
		_, err := sm.client.UploadFile(params)
		if err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}
		return nil
	}

	var options = []slack.MsgOption{slack.MsgOptionText(formatx.CodeBlock(sm.Response), false), slack.MsgOptionAsUser(true)}

	//if the message is from thread then add an option to return the response to the thread
	if sm.Event.ThreadTimeStamp != "" {
		options = append(options, slack.MsgOptionTS(sm.Event.ThreadTimeStamp))
	}

	if _, _, err := sm.client.PostMessage(sm.Event.Channel, options...); err != nil {
		return fmt.Errorf("while posting Slack message: %w", err)
	}

	return nil
}

// SendEvent sends event notification to slack
func (b *Slack) SendEvent(ctx context.Context, event events.Event, eventSources []string) error {
	b.log.Debugf(">> Sending to Slack: %+v", event)
	attachment := b.formatMessage(event)

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
			b.log.Info("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
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
		b.log.Debugf(">> Sending message to channel %q: %+v", channelName, msg)
		channelID, timestamp, err := b.client.PostMessageContext(ctx, channelName, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q: %w", channelName, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %b at %b", channelID, timestamp)
	}

	return errs.ErrorOrNil()
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
			notify:                defaultNotifyValue,
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
