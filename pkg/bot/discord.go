// Copyright (c) 2020 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
)

// DiscordBot listens for user's message, execute commands and sends back the response
type DiscordBot struct {
	log              logrus.FieldLogger
	executorFactory  ExecutorFactory
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
	Event           *discordgo.MessageCreate
	BotID           string
	Request         string
	Response        string
	IsAuthChannel   bool
	Session         *discordgo.Session
}

// NewDiscordBot returns new Bot object
func NewDiscordBot(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory) *DiscordBot {
	return &DiscordBot{
		log:              log,
		executorFactory:  executorFactory,
		Token:            c.Communications.Discord.Token,
		BotID:            c.Communications.Discord.BotID,
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelID:        c.Communications.Discord.Channel,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
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

	b.log.Info("BotKube connected to Discord!")

	<-ctx.Done()
	b.log.Info("Shutdown requested. Finishing...")
	err = api.Close()
	if err != nil {
		return fmt.Errorf("while closing connection: %w", err)
	}

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

	e := dm.executorFactory.NewDefault(config.DiscordBot, dm.IsAuthChannel, dm.Request)

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
