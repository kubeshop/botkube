package bot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//    - handle and send methods from `slackMessage` should be defined on Bot level,
//    - split to multiple files in a separate package,
//    - review all the methods and see if they can be simplified.

// slackMaxMessageSize max size before a message should be uploaded as a file.
//
// "The text for the block, in the form of a text object.
//
//	Maximum length for the text in this field is 3000 characters.  (..)"
//
// source: https://api.slack.com/reference/block-kit/blocks#section
const slackMaxMessageSize = 3001

var _ Bot = &Slack{}

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

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	Text            string
	Channel         string
	ThreadTimeStamp string
	User            string
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
		channels:        channels,
		commGroupName:   commGroupName,
		botMentionRegex: botMentionRegex,
		renderer:        NewSlackRenderer(),
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

				b.log.Info("Botkube connected to Slack!")

			case *slack.MessageEvent:
				// Skip if message posted by Botkube
				if ev.User == b.botID {
					continue
				}
				sm := slackMessage{
					Text:            ev.Text,
					Channel:         ev.Channel,
					ThreadTimeStamp: ev.ThreadTimestamp,
					User:            ev.User,
				}
				err := b.handleMessage(ctx, sm)
				if err != nil {
					wrappedErr := fmt.Errorf("while handling message: %w", err)
					b.log.Errorf(wrappedErr.Error())
				}

			case *slack.RTMError:
				b.log.Errorf("Slack RTM error: %+v", ev.Error())

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

func (b *Slack) handleMessage(ctx context.Context, msg slackMessage) error {
	// Handle message only if starts with mention
	request, found := b.findAndTrimBotMention(msg.Text)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}

	b.log.Debugf("Slack incoming Request: %s", request)

	// Unfortunately we need to do a call for channel name based on ID every time a message arrives.
	// I wanted to query for channel IDs based on names and prepare a map in the `slackChannelsConfigFrom`,
	// but unfortunately Botkube would need another scope (get all conversations).
	// Keeping current way of doing this until we come up with a better idea.
	info, err := b.client.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID:     msg.Channel,
		IncludeLocale: true,
	})
	if err != nil {
		return fmt.Errorf("while getting conversation info: %w", err)
	}

	channel, isAuthChannel := b.getChannels()[info.Name]

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			ID:               channel.Identifier(),
			ExecutorBindings: channel.Bindings.Executors,
			IsAuthenticated:  isAuthChannel,
			CommandOrigin:    command.TypedOrigin,
		},
		Message: request,
		User: execute.UserInput{
			Mention:     fmt.Sprintf("<@%s>", msg.User),
			DisplayName: msg.User, // this integration is officially not supported, so no need to ensure it has a nice display name
		},
	})
	response := e.Execute(ctx)
	err = b.send(ctx, msg, response, response.OnlyVisibleForYou)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *Slack) send(ctx context.Context, msg slackMessage, resp interactive.CoreMessage, onlyVisibleToUser bool) error {
	b.log.Debugf("Sending message to channel %q: %+v", msg.Channel, msg)

	resp.ReplaceBotNamePlaceholder(b.BotName())
	markdown := b.renderer.MessageToMarkdown(resp)

	if len(markdown) == 0 {
		return errors.New("while reading Slack response: empty response")
	}

	// Upload message as a file if too long
	if len(markdown) >= slackMaxMessageSize {
		_, err := uploadFileToSlack(ctx, msg.Channel, resp, b.client, msg.ThreadTimeStamp)
		if err != nil {
			return err
		}
		return nil
	}

	var options = []slack.MsgOption{slack.MsgOptionText(markdown, false), slack.MsgOptionAsUser(true)}

	//if the message is from thread then add an option to return the response to the thread
	if msg.ThreadTimeStamp != "" {
		options = append(options, slack.MsgOptionTS(msg.ThreadTimeStamp))
	}

	if onlyVisibleToUser {
		if _, err := b.client.PostEphemeralContext(ctx, msg.Channel, msg.User, options...); err != nil {
			return fmt.Errorf("while posting Slack message visible only to user: %w", err)
		}
	} else {
		if _, _, err := b.client.PostMessageContext(ctx, msg.Channel, options...); err != nil {
			return fmt.Errorf("while posting Slack message: %w", err)
		}
	}

	b.log.Debugf("Message successfully sent to channel %q", msg.Channel)
	return nil
}

func (b *Slack) getChannelsToNotify(sourceBindings []string) []string {
	var out []string
	for _, cfg := range b.getChannels() {
		if !cfg.notify {
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
			continue
		}

		if !sliceutil.Intersect(sourceBindings, cfg.Bindings.Sources) {
			continue
		}

		out = append(out, cfg.Identifier())
	}
	return out
}

// SendMessage sends message to selected Slack channels.
func (b *Slack) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	errs := multierror.New()
	for _, channelName := range b.getChannelsToNotify(sourceBindings) {
		msgMetadata := slackMessage{
			Channel:         channelName,
			ThreadTimeStamp: "",
		}
		err := b.send(ctx, msgMetadata, msg, false)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q: %w", channelName, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends message to all Slack channels.
func (b *Slack) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelName := channel.Name
		msgMetadata := slackMessage{
			Channel:         channelName,
			ThreadTimeStamp: "",
		}
		err := b.send(ctx, msgMetadata, msg, false)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q (alias: %q): %w", channelName, channel.alias, err))
			continue
		}
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

func uploadFileToSlack(ctx context.Context, channel string, resp interactive.CoreMessage, client *slack.Client, ts string) (*slack.File, error) {
	params := slack.FileUploadParameters{
		Filename:        "Response.txt",
		Title:           "Response.txt",
		InitialComment:  resp.Description,
		Content:         interactive.MessageToPlaintext(resp, interactive.NewlineFormatter),
		Channels:        []string{channel},
		ThreadTimestamp: ts,
	}

	file, err := client.UploadFileContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("while uploading file: %w", err)
	}

	return file, nil
}
