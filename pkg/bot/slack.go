package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	format2 "github.com/kubeshop/botkube/pkg/format"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// TODO: Refactor:
// 	- handle and send methods from `slackMessage` should be defined on Bot level,
//  - split to multiple files in a separate package,
//  - review all the methods and see if they can be simplified.

var _ Bot = &Slack{}

const sendFailureMessageFmt = "Unable to send message to ChannelName `%s`: `%s`\n```add Botkube app to the ChannelName %s\nMissed events follows below:```"
const channelNotFoundCode = "channel_not_found"

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
	notifyMutex     sync.RWMutex
	notify          bool
	botID           string

	Client           *slack.Client
	Notification     config.Notification
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelName      string
	DefaultNamespace string
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	Event         *slack.MessageEvent
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	RTM           *slack.RTM
}

// NewSlack creates a new Slack instance.
func NewSlack(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter FatalErrorAnalyticsReporter) (*Slack, error) {
	slackCfg := c.Communications.GetFirst().Slack

	client := slack.New(slackCfg.Token)

	authResp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("while testing the ability to do auth Slack request: %w", err)
	}
	botID := authResp.UserID

	return &Slack{
		log:              log,
		executorFactory:  executorFactory,
		reporter:         reporter,
		notify:           true, // enabled by default
		botID:            botID,
		Client:           client,
		Notification:     slackCfg.Notification,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelName:      slackCfg.Channels.GetFirst().Name,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
	}, nil
}

// Start starts the Slack RTM connection and listens for messages
func (b *Slack) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	rtm := b.Client.NewRTM()
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
				err := sm.HandleMessage(b)
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

// Enabled returns current notification status.
func (b *Slack) Enabled() bool {
	b.notifyMutex.RLock()
	defer b.notifyMutex.RUnlock()
	return b.notify
}

// SetEnabled sets a new notification status.
func (b *Slack) SetEnabled(value bool) error {
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()
	b.notify = value
	return nil
}

func (sm *slackMessage) HandleMessage(b *Slack) error {
	// Check if message posted in authenticated channel
	info, err := b.Client.GetConversationInfo(sm.Event.Channel, true)
	if err == nil {
		if info.IsChannel || info.IsPrivate {
			// Message posted in a channel
			// Serve only if starts with mention
			if !strings.HasPrefix(sm.Event.Text, "<@"+sm.BotID+">") {
				sm.log.Debugf("Ignoring message as it doesn't contain %q prefix", sm.BotID)
				return nil
			}
			// Serve only if current channel is in config
			if b.ChannelName == info.Name {
				sm.IsAuthChannel = true
			}
		}
	}
	// Serve only if current channel is in config
	if b.ChannelName == sm.Event.Channel {
		sm.IsAuthChannel = true
	}

	// Trim the @BotKube prefix
	sm.Request = strings.TrimPrefix(sm.Event.Text, "<@"+sm.BotID+">")

	e := sm.executorFactory.NewDefault(b.IntegrationName(), b, sm.IsAuthChannel, sm.Request)
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
		_, err := sm.RTM.UploadFile(params)
		if err != nil {
			return fmt.Errorf("while uploading file: %w", err)
		}
		return nil
	}

	var options = []slack.MsgOption{slack.MsgOptionText(format2.CodeBlock(sm.Response), false), slack.MsgOptionAsUser(true)}

	//if the message is from thread then add an option to return the response to the thread
	if sm.Event.ThreadTimestamp != "" {
		options = append(options, slack.MsgOptionTS(sm.Event.ThreadTimestamp))
	}

	if _, _, err := sm.RTM.PostMessage(sm.Event.Channel, options...); err != nil {
		return fmt.Errorf("while posting Slack message: %w", err)
	}

	return nil
}

// SendEvent sends event notification to slack
func (b *Slack) SendEvent(ctx context.Context, event events.Event) error {
	if !b.notify {
		b.log.Info("Notifications are disabled. Skipping event...")
		return nil
	}

	b.log.Debugf(">> Sending to slack: %+v", event)
	attachment := b.formatMessage(event, b.Notification)

	targetChannel := event.Channel
	if targetChannel == "" {
		// empty value in event.channel sends notifications to default channel.
		targetChannel = b.ChannelName
	}
	isDefaultChannel := targetChannel == b.ChannelName

	channelID, timestamp, err := b.Client.PostMessage(targetChannel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
	if err != nil {
		postMessageWrappedErr := fmt.Errorf("while posting message to channel %q: %w", targetChannel, err)

		if isDefaultChannel || err.Error() != channelNotFoundCode {
			return postMessageWrappedErr
		}

		// channel not found, fallback to default channel

		// send error message to default channel
		msg := fmt.Sprintf(sendFailureMessageFmt, targetChannel, err.Error(), targetChannel)
		sendMessageErr := b.SendMessage(ctx, msg)
		if sendMessageErr != nil {
			return multierror.Append(postMessageWrappedErr, sendMessageErr)
		}

		// sending missed event to default channel
		// reset event.ChannelName and send event
		event.Channel = ""
		sendEventErr := b.SendEvent(ctx, event)
		if sendEventErr != nil {
			return multierror.Append(postMessageWrappedErr, sendEventErr)
		}

		return postMessageWrappedErr
	}

	b.log.Debugf("Event successfully sent to channel %q at %b", channelID, timestamp)
	return nil
}

// SendMessage sends message to slack channel
func (b *Slack) SendMessage(ctx context.Context, msg string) error {
	b.log.Debugf(">> Sending to slack: %+v", msg)
	channelID, timestamp, err := b.Client.PostMessageContext(ctx, b.ChannelName, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		return fmt.Errorf("while sending Slack message to channel %q: %w", b.ChannelName, err)
	}

	b.log.Debugf("Message successfully sent to channel %b at %b", channelID, timestamp)
	return nil
}
