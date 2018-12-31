package slack

import (
	"fmt"
	"strings"

	"github.com/infracloudio/kubeops/pkg/config"
	"github.com/infracloudio/kubeops/pkg/logging"
	"github.com/nlopes/slack"
)

// Bot listens for user's message, execute commands and sends back the response
type Bot struct {
	Token string
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	ChannelID    string
	BotID        string
	InMessage    string
	OutMessage   string
	OutMsgLength int
	RTM          *slack.RTM
}

// NewSlackBot returns new Bot object
func NewSlackBot() *Bot {
	c, err := config.New()
	if err != nil {
		logging.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	return &Bot{
		Token: c.Communications.Slack.Token,
	}
}

// Start starts the slacknot RTM connection and listens for messages
func (s *Bot) Start() {
	api := slack.New(s.Token)
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
			// Serve only if mentioned
			if !strings.HasPrefix(ev.Text, "<@"+botID+"> ") {
				continue
			}
			logging.Logger.Debugf("Slack incoming message: %+v", ev)
			msg := strings.TrimPrefix(ev.Text, "<@"+botID+"> ")
			sm := slackMessage{
				ChannelID: ev.Channel,
				BotID:     botID,
				InMessage: msg,
				RTM:       rtm,
			}
			sm.HandleMessage()

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return
		default:
		}
	}
}

func (sm *slackMessage) HandleMessage() {
	sm.OutMessage = parseAndRunCommand(sm.InMessage)
	sm.OutMsgLength = len(sm.OutMessage)
	sm.Send()
}

func formatAndSendLogs(rtm *slack.RTM, channelID, logs string, filename string) {
	params := slack.FileUploadParameters{
		Title:    filename,
		Content:  logs,
		Filetype: "log",
		Channels: []string{channelID},
	}
	_, err := rtm.UploadFile(params)
	if err != nil {
		logging.Logger.Error("Error in uploading file:", err)
	}
}

func (sm slackMessage) Send() {
	// Upload message as a file if too long
	if sm.OutMsgLength >= 3990 {
		params := slack.FileUploadParameters{
			Title:    sm.InMessage,
			Content:  sm.OutMessage,
			Channels: []string{sm.ChannelID},
		}
		_, err := sm.RTM.UploadFile(params)
		if err != nil {
			logging.Logger.Error("Error in uploading file:", err)
		}
		return
	}
	params := slack.PostMessageParameters{
		AsUser: true,
	}
	_, _, err := sm.RTM.PostMessage(sm.ChannelID, "```"+sm.OutMessage+"```", params)
	if err != nil {
		logging.Logger.Error("Error in sending message:", err)
	}
}
