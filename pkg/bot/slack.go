package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
)

var _ Bot = &SlackBot{}

const sendFailureMessageFmt = "Unable to send message to ChannelName `%s`: `%s`\n```add Botkube app to the ChannelName %s\nMissed events follows below:```"
const channelNotFoundCode = "channel_not_found"

var attachmentColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "danger",
	config.Critical: "danger",
}

// SlackBot listens for user's message, execute commands and sends back the response
type SlackBot struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        FatalErrorAnalyticsReporter
	notify          bool

	Client           *slack.Client
	Notification     config.Notification
	Token            string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelName      string
	SlackURL         string
	BotID            string
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

// NewSlackBot returns new Bot object
func NewSlackBot(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter FatalErrorAnalyticsReporter) *SlackBot {
	slackCfg := c.Communications.GetFirst().Slack
	return &SlackBot{
		log:              log,
		executorFactory:  executorFactory,
		reporter:         reporter,
		notify:           true, // enabled by default
		Token:            slackCfg.Token,
		Client:           slack.New(slackCfg.Token),
		Notification:     slackCfg.Notification,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelName:      slackCfg.Channels.GetFirst().Name,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
	}
}

// Start starts the Slack RTM connection and listens for messages
func (b *SlackBot) Start(ctx context.Context) error {
	b.log.Info("Starting bot")
	var botID string
	api := slack.New(b.Token)
	if len(b.SlackURL) != 0 {
		api = slack.New(b.Token, slack.OptionAPIURL(b.SlackURL))
		botID = b.BotID
	} else {
		authResp, err := api.AuthTest()
		if err != nil {
			return fmt.Errorf("while testing the ability to do auth request: %w", err)
		}
		botID = authResp.UserID
	}

	rtm := api.NewRTM()
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
				if ev.User == botID {
					continue
				}
				sm := slackMessage{
					log:             b.log,
					executorFactory: b.executorFactory,
					Event:           ev,
					BotID:           botID,
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

// Type describes the sink type.
func (b *SlackBot) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// IntegrationName describes the sink integration name.
func (b *SlackBot) IntegrationName() config.CommPlatformIntegration {
	return config.SlackCommPlatformIntegration
}

// Enabled returns current notification status.
func (b *SlackBot) Enabled() bool {
	return b.notify
}

// SetEnabled sets a new notification status.
func (b *SlackBot) SetEnabled(value bool) error {
	b.notify = value
	return nil
}

// TODO: refactor - handle and send methods should be defined on Bot level

func (sm *slackMessage) HandleMessage(b *SlackBot) error {
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

	var options = []slack.MsgOption{slack.MsgOptionText(formatCodeBlock(sm.Response), false), slack.MsgOptionAsUser(true)}

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
func (b *SlackBot) SendEvent(ctx context.Context, event events.Event) error {
	b.log.Debugf(">> Sending to slack: %+v", event)
	attachment := b.formatSlackMessage(event, b.Notification)

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
func (b *SlackBot) SendMessage(ctx context.Context, msg string) error {
	b.log.Debugf(">> Sending to slack: %+v", msg)
	channelID, timestamp, err := b.Client.PostMessageContext(ctx, b.ChannelName, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		return fmt.Errorf("while sending SlackBot message to channel %q: %w", b.ChannelName, err)
	}

	b.log.Debugf("Message successfully sent to channel %b at %b", channelID, timestamp)
	return nil
}

func (b *SlackBot) formatSlackMessage(event events.Event, notification config.Notification) (attachment slack.Attachment) {
	switch notification.Type {
	case config.LongNotification:
		attachment = b.slackLongNotification(event)

	case config.ShortNotification:
		fallthrough

	default:
		// set missing cluster name to event object
		attachment = b.slackShortNotification(event)
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]
	return attachment
}

func (b *SlackBot) slackLongNotification(event events.Event) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*%s*", event.Title),
		Fields: []slack.AttachmentField{
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
		},
		Footer: "BotKube",
	}
	if event.Namespace != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Namespace",
			Value: event.Namespace,
			Short: true,
		})
	}

	if event.Reason != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
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
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Warnings",
			Value: warn,
		})
	}

	// Add clusterName in the message
	attachment.Fields = append(attachment.Fields, slack.AttachmentField{
		Title: "Cluster",
		Value: event.Cluster,
	})
	return attachment
}

func (b *SlackBot) slackShortNotification(event events.Event) slack.Attachment {
	return slack.Attachment{
		Title: event.Title,
		Fields: []slack.AttachmentField{
			{
				Value: FormatShortMessage(event),
			},
		},
		Footer: "BotKube",
	}
}
