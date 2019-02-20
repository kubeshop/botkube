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
	ChannelID    string
	BotID        string
	MessageType  string
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
		isAuthChannel := false
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			logging.Logger.Debug("Connection Info: ", ev.Info)

		case *slack.MessageEvent:
			// Skip if message posted by BotKube
			if ev.User == botID {
				continue
			}

			info, err := api.GetConversationInfo(ev.Channel, true)
			if err == nil {
				if info.IsChannel || info.IsPrivate {
					// Message posted in a channel
					// Serve only if starts with mention
					if !strings.HasPrefix(ev.Text, "<@"+botID+"> ") {
						continue
					}
					// Serve only if current channel is in config
					if b.ChannelName == info.Name {
						isAuthChannel = true
					}
				}
			}

			// Message posted as a DM
			logging.Logger.Debugf("Slack incoming message: %+v", ev)
			inMessage := ev.Text

			// Trim @BotKube prefix if exists
			if strings.HasPrefix(ev.Text, "<@"+botID+"> ") {
				inMessage = strings.TrimPrefix(ev.Text, "<@"+botID+"> ")
			}
			sm := slackMessage{
				ChannelID: ev.Channel,
				BotID:     botID,
				InMessage: inMessage,
				RTM:       rtm,
			}
			sm.HandleMessage(b.AllowKubectl, b.ClusterName, b.ChannelName, isAuthChannel)

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return
		default:
		}
	}
}

func (sm *slackMessage) HandleMessage(allowkubectl bool, clusterName, channelName string, isAuthChannel bool) {
	e := execute.NewDefaultExecutor(sm.InMessage, allowkubectl, clusterName, channelName, isAuthChannel)
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
	} else if sm.OutMsgLength == 0 {
		logging.Logger.Info("Invalid request. Dumping the response")
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
