package bot

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
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
	commGroupName   string
	renderer        *SlackRenderer
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	Event    *slack.MessageEvent
	BotID    string
	Request  string
	Response string
	RTM      *slack.RTM
}

// NewSlack creates a new Slack instance.
func NewSlack(log logrus.FieldLogger, commGroupName string, cfg config.Slack, executorFactory ExecutorFactory, reporter FatalErrorAnalyticsReporter) (*Slack, error) {
	client := slack.New(cfg.Token)

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
		commGroupName:   commGroupName,
		renderer:        NewSlackRenderer(cfg.Notification),
		botMentionRegex: botMentionRegex,
	}, nil
}

// Start starts the Slack RTM connection and listens for messages
func (b *Slack) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	rtm := b.client.NewRTM()
	go func() {
		defer analytics.ReportPanicIfOccurs(b.log, b.reporter)
		rtm.ManageConnection()
	}()

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			return rtm.Disconnect()
		case msg, ok := <-rtm.IncomingEvents:
			if !ok {
				b.log.Info("Incoming events channel closed. Finishing...")
				return nil
			}

			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				err := b.reporter.ReportBotEnabled(b.IntegrationName())
				if err != nil {
					return fmt.Errorf("while reporting analytics: %w", err)
				}

				b.log.Info("BotKube connected to Slack!")

			case *slack.MessageEvent:
				// Skip if message posted by BotKube
				if ev.User == b.botID {
					continue
				}
				sm := slackMessage{
					log:             b.log,
					executorFactory: b.executorFactory,
					Event:           ev,
					BotID:           b.botID,
					RTM:             rtm,
				}
				err := b.handleMessage(sm)
				if err != nil {
					wrappedErr := fmt.Errorf("while handling message: %w", err)
					b.log.Errorf(wrappedErr.Error())
				}

			case *slack.RTMError:
				b.log.Errorf("Slack RMT error: %+v", ev.Error())

			case *slack.ConnectionErrorEvent:
				b.log.Errorf("Slack connection error: %+v", ev.Error())

			case *slack.IncomingEventError:
				b.log.Errorf("Slack incoming event error: %+v", ev.Error())

			case *slack.OutgoingErrorEvent:
				b.log.Errorf("Slack outgoing event error: %+v", ev.Error())

			case *slack.UnmarshallingErrorEvent:
				b.log.Warningf("Slack unmarshalling error: %+v", ev.Error())

			case *slack.RateLimitedError:
				b.log.Errorf("Slack rate limiting error: %+v", ev.Error())

			case *slack.InvalidAuthEvent:
				return fmt.Errorf("invalid credentials")
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

func (b *Slack) handleMessage(msg slackMessage) error {
	// Handle message only if starts with mention
	request, found := b.findAndTrimBotMention(msg.Event.Text)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}

	// Unfortunately we need to do a call for channel name based on ID every time a message arrives.
	// I wanted to query for channel IDs based on names and prepare a map in the `slackChannelsConfigFrom`,
	// but unfortunately BotKube would need another scope (get all conversations).
	// Keeping current way of doing this until we come up with a better idea.
	info, err := b.client.GetConversationInfo(msg.Event.Channel, true)
	if err != nil {
		return fmt.Errorf("while getting conversation info: %w", err)
	}

	channel, isAuthChannel := b.getChannels()[info.Name]

	e := b.executorFactory.NewDefault(b.commGroupName, b.IntegrationName(), b, isAuthChannel, info.Name, channel.Bindings.Executors, request)
	response := e.Execute()
	err = b.send(msg, request, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *Slack) send(msg slackMessage, req string, resp interactive.Message) error {
	b.log.Debugf("Slack incoming Request: %s", req)
	b.log.Debugf("Slack Response: %s", resp)

	plaintext := interactive.MessageToMarkdown(interactive.MDLineFmt, resp)

	if len(plaintext) == 0 {
		return fmt.Errorf("while reading Slack response: empty response for request %q", req)
	}
	// Upload message as a file if too long
	if len(plaintext) >= 3990 {
		params := slack.FileUploadParameters{
			Filename: req,
			Title:    req,
			Content:  plaintext,
			Channels: []string{msg.Event.Channel},
		}
		_, err := msg.RTM.UploadFile(params)
		if err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}
		return nil
	}

	var options = []slack.MsgOption{slack.MsgOptionText(formatx.CodeBlock(plaintext), false), slack.MsgOptionAsUser(true)}

	//if the message is from thread then add an option to return the response to the thread
	if msg.Event.ThreadTimestamp != "" {
		options = append(options, slack.MsgOptionTS(msg.Event.ThreadTimestamp))
	}

	if _, _, err := msg.RTM.PostMessage(msg.Event.Channel, options...); err != nil {
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
func (b *Slack) SendMessage(ctx context.Context, msg interactive.Message) error {
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
