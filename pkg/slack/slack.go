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
		Token:        c.Communications.Slack.Token,
		AllowKubectl: c.Settings.AllowKubectl,
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
			// Check if message posted in a channel
			chanInfo, _ := api.GetChannelInfo(ev.Channel)

			// Check if message posted in a group
			groupInfo, _ := api.GetGroupInfo(ev.Channel)

			if chanInfo != nil || groupInfo != nil {
				// Message posted in a channel/group
				// Serve only if starts with mention
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
				sm.HandleMessage(b.AllowKubectl)
				continue
			}

			// Message posted as a DM
			// Skip if message posted by BotKube
			if ev.User == botID {
				continue
			}
			logging.Logger.Debugf("Slack incoming message: %+v", ev)
			msg := ev.Text

			// Trim @BotKube prefix if exists
			if strings.HasPrefix(ev.Text, "<@"+botID+"> ") {
				msg = strings.TrimPrefix(ev.Text, "<@"+botID+"> ")
			}
			sm := slackMessage{
				ChannelID: ev.Channel,
				BotID:     botID,
				InMessage: msg,
				RTM:       rtm,
			}
			sm.HandleMessage(b.AllowKubectl)

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return
		default:
		}
	}
}

func (sm *slackMessage) HandleMessage(allowkubectl bool) {
	e := execute.NewDefaultExecutor(sm.InMessage, allowkubectl)
	sm.OutMessage = e.Execute()
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
