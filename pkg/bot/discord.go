package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

var _ Bot = &DiscordBot{}

// DiscordBot listens for user's message, execute commands and sends back the response
type DiscordBot struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	notify          bool

	Token            string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelID        string
	BotID            string
	DefaultNamespace string
}

// discordMessage contains message details to execute command and send back the result
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

// NewDiscordBot returns new Bot object
func NewDiscordBot(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory, reporter AnalyticsReporter) *DiscordBot {
	discord := c.Communications.GetFirst().Discord
	return &DiscordBot{
		log:             log,
		reporter:        reporter,
		executorFactory: executorFactory,
		notify:          true, // enabled by default

		Token:            discord.Token,
		BotID:            discord.BotID,
		AllowKubectl:     c.Executors.GetFirst().Kubectl.Enabled,
		RestrictAccess:   c.Executors.GetFirst().Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelID:        discord.Channels.GetFirst().ID,
		DefaultNamespace: c.Executors.GetFirst().Kubectl.DefaultNamespace,
	}
}

// Start starts the DiscordBot websocket connection and listens for messages
func (b *DiscordBot) Start(ctx context.Context) error {
	b.log.Info("Starting bot")
	api, err := discordgo.New("Bot " + b.Token)
	if err != nil {
		return fmt.Errorf("while creating Discord session: %w", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	api.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
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
	err = api.Open()
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
	err = api.Close()
	if err != nil {
		return fmt.Errorf("while closing connection: %w", err)
	}

	return nil
}

// IntegrationName describes the integration name.
func (b *DiscordBot) IntegrationName() config.CommPlatformIntegration {
	return config.DiscordCommPlatformIntegration
}

// Type describes the integration type.
func (b *DiscordBot) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// Enabled returns current notification status.
func (b *DiscordBot) Enabled() bool {
	return b.notify
}

// SetEnabled sets a new notification status.
func (b *DiscordBot) SetEnabled(value bool) error {
	b.notify = value
	return nil
}

// TODO: refactor - handle and send methods should be defined on Bot level

// HandleMessage handles the incoming messages
func (dm *discordMessage) HandleMessage(b *DiscordBot) {
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

	if _, err := dm.Session.ChannelMessageSend(dm.Event.ChannelID, formatCodeBlock(dm.Response)); err != nil {
		dm.log.Error("Error in sending message:", err)
	}
}
