package slack

import (
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/logging"
	"github.com/nlopes/slack"
)

// Bot listens for user's message, execute commands and sends back the response
type Bot struct {
	Token        string
	AllowKubectl bool
	ClusterName  string
	ChannelName  string
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	Event         *slack.MessageEvent
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	RTM           *slack.RTM
}

// NewSlackBot returns new Bot object
func NewSlackBot() *Bot {
	c, err := config.New()
	if err != nil {
		logging.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	return &Bot{
		Token:        c.Communications.Slack.Token,
		AllowKubectl: c.Settings.AllowKubectl,
		ClusterName:  c.Settings.ClusterName,
		ChannelName:  c.Communications.Slack.Channel,
	}
}

// Start starts the slacknot RTM connection and listens for messages
func (b *Bot) Start() {
	api := slack.New(b.Token)
	authResp, err := api.AuthTest()
	if err != nil {
		logging.Logger.Fatal(err)
	}
	botID := authResp.UserID

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			logging.Logger.Debug("Connection Info: ", ev.Info)

		case *slack.MessageEvent:
			// Skip if message posted by BotKube
			if ev.User == botID {
				continue
			}
			sm := slackMessage{
				Event: ev,
				BotID: botID,
				RTM:   rtm,
			}
			sm.HandleMessage(b)

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return
		default:
		}
	}
}

func (sm *slackMessage) HandleMessage(b *Bot) {
	logging.Logger.Debugf("Slack incoming message: %+v", sm.Event)
	// Check if message posted in authenticated channel
	info, err := slack.New(b.Token).GetConversationInfo(sm.Event.Channel, true)
	if err == nil {
		if info.IsChannel || info.IsPrivate {
			// Message posted in a channel
			// Serve only if starts with mention
			if !strings.HasPrefix(sm.Event.Text, "<@"+sm.BotID+"> ") {
				return
			}
			// Serve only if current channel is in config
			if b.ChannelName == info.Name {
				sm.IsAuthChannel = true
			}
		}
	}

	// Trim the @BotKube prefix
	sm.Request = strings.TrimPrefix(sm.Event.Text, "<@"+sm.BotID+"> ")

	e := execute.NewDefaultExecutor(sm.Request, b.AllowKubectl, b.ClusterName, b.ChannelName, sm.IsAuthChannel)
	sm.Response = e.Execute()
	sm.Send()
}

func (sm slackMessage) Send() {
	// Upload message as a file if too long
	if len(sm.Response) >= 3990 {
		params := slack.FileUploadParameters{
			Title:    sm.Request,
			Content:  sm.Response,
			Channels: []string{sm.Event.Channel},
		}
		_, err := sm.RTM.UploadFile(params)
		if err != nil {
			logging.Logger.Error("Error in uploading file:", err)
		}
		return
	} else if len(sm.Response) == 0 {
		logging.Logger.Info("Invalid request. Dumping the response")
		return
	}

	params := slack.PostMessageParameters{
		AsUser: true,
	}
	if _, _, err := sm.RTM.PostMessage(sm.Event.Channel, "```"+sm.Response+"```", params); err != nil {
		logging.Logger.Error("Error in sending message:", err)
	}
}
