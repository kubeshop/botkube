package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/format"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//    - handle and send methods from `discordMessage` should be defined on Bot level,
//    - split to multiple files in a separate package,
//    - review all the methods and see if they can be simplified.

var _ Bot = &Discord{}

// customTimeFormat holds custom time format string.
const customTimeFormat = "2006-01-02T15:04:05Z"

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
	notifyMutex     sync.RWMutex
	notify          bool
	api             *discordgo.Session

	Notification     config.Notification
	Token            string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelID        string
	BotID            string
	DefaultNamespace string
}

// discordMessage contains message details to execute command and send back the result.
type discordMessage struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	Event         *discordgo.MessageCreate
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	Session       *discordgo.Session
}

// NewDiscord creates a new Discord instance.
func NewDiscord(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Discord, error) {
	discord := c.Communications.GetFirst().Discord

	api, err := discordgo.New("Bot " + discord.Token)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}

	return &Discord{
		log:             log,
		reporter:        reporter,
		executorFactory: executorFactory,
		notify:          true, // enabled by default
		api:             api,

		Token:            discord.Token,
		BotID:            discord.BotID,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelID:        discord.Channels.GetFirst().ID,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
		Notification:     discord.Notification,
	}, nil
}

// Start starts the Discord websocket connection and listens for messages.
func (b *Discord) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	// Register the messageCreate func as a callback for MessageCreate events.
	b.api.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		dm := discordMessage{
			log:             b.log,
			executorFactory: b.executorFactory,
			Event:           m,
			BotID:           b.BotID,
			Session:         s,
		}

		dm.HandleMessage(b)
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

	b.log.Info("BotKube connected to Discord!")

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
func (b *Discord) SendEvent(_ context.Context, event events.Event) (err error) {
	if !b.notify {
		b.log.Info("Notifications are disabled. Skipping event...")
		return nil
	}

	b.log.Debugf(">> Sending to Discord: %+v", event)

	messageSend := b.formatMessage(event, b.Notification)

	if _, err := b.api.ChannelMessageSendComplex(b.ChannelID, &messageSend); err != nil {
		return fmt.Errorf("while sending Discord message to channel %q: %w", b.ChannelID, err)
	}

	b.log.Debugf("Event successfully sent to channel %s", b.ChannelID)
	return nil
}

// SendMessage sends message to Discord ChannelName.
// Context is not supported by client: See https://github.com/bwmarrin/discordgo/issues/752.
func (b *Discord) SendMessage(_ context.Context, msg string) error {
	b.log.Debugf(">> Sending to Discord: %+v", msg)

	if _, err := b.api.ChannelMessageSend(b.ChannelID, msg); err != nil {
		return fmt.Errorf("while sending Discord message to channel %q: %w", b.ChannelID, err)
	}
	b.log.Debugf("Event successfully sent to Discord %v", msg)
	return nil
}

// IntegrationName describes the integration name.
func (b *Discord) IntegrationName() config.CommPlatformIntegration {
	return config.DiscordCommPlatformIntegration
}

// Type describes the integration type.
func (b *Discord) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// NotificationsEnabled returns current notification status.
func (b *Discord) NotificationsEnabled() bool {
	b.notifyMutex.RLock()
	defer b.notifyMutex.RUnlock()
	return b.notify
}

// SetNotificationsEnabled sets a new notification status.
func (b *Discord) SetNotificationsEnabled(enabled bool) error {
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()
	b.notify = enabled
	return nil
}

// HandleMessage handles the incoming messages.
func (dm *discordMessage) HandleMessage(b *Discord) {
	// Serve only if starts with mention
	if !strings.HasPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ") && !strings.HasPrefix(dm.Event.Content, "<@"+dm.BotID+"> ") {
		return
	}

	// Serve only if current channel is in config
	if b.ChannelID == dm.Event.ChannelID {
		dm.IsAuthChannel = true
	}

	// Trim the @BotKube prefix
	if strings.HasPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ") {
		dm.Request = strings.TrimPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ")
	} else if strings.HasPrefix(dm.Event.Content, "<@"+dm.BotID+"> ") {
		dm.Request = strings.TrimPrefix(dm.Event.Content, "<@"+dm.BotID+"> ")
	}

	if len(dm.Request) == 0 {
		return
	}

	e := dm.executorFactory.NewDefault(b.IntegrationName(), b, dm.IsAuthChannel, dm.Request)

	dm.Response = e.Execute()
	dm.Send()
}

func (dm discordMessage) Send() {
	dm.log.Debugf("Discord incoming Request: %s", dm.Request)
	dm.log.Debugf("Discord Response: %s", dm.Response)

	if len(dm.Response) == 0 {
		dm.log.Errorf("Invalid request. Dumping the response. Request: %s", dm.Request)
		return
	}

	// Upload message as a file if too long
	if len(dm.Response) >= 2000 {
		params := &discordgo.MessageSend{
			Content: dm.Request,
			Files: []*discordgo.File{
				{
					Name:   "Response",
					Reader: strings.NewReader(dm.Response),
				},
			},
		}
		if _, err := dm.Session.ChannelMessageSendComplex(dm.Event.ChannelID, params); err != nil {
			dm.log.Error("Error in uploading file:", err)
		}
		return
	}

	if _, err := dm.Session.ChannelMessageSend(dm.Event.ChannelID, format.CodeBlock(dm.Response)); err != nil {
		dm.log.Error("Error in sending message:", err)
	}
}
