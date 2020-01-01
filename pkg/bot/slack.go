package bot

import (
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/logging"
	"github.com/nlopes/slack"
)

// SlackBot listens for user's message, execute commands and sends back the response
type SlackBot struct {
	Token          string
	AllowKubectl   bool
	RestrictAccess bool
	ClusterName    string
	ChannelName    string
	SlackURL       string
	BotID          string
}

// slackMessage contains message details to execute command and send back the result
type slackMessage struct {
	Event         *slack.MessageEvent
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	RTM           *slack.RTM
	SlackClient   *slack.Client
}

// NewSlackBot returns new Bot object
func NewSlackBot() Bot {
	c, err := config.New()
	if err != nil {
		logging.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	return &SlackBot{
		Token:          c.Communications.Slack.Token,
		AllowKubectl:   c.Settings.AllowKubectl,
		RestrictAccess: c.Settings.RestrictAccess,
		ClusterName:    c.Settings.ClusterName,
		ChannelName:    c.Communications.Slack.Channel,
	}
}

// Start starts the slacknot RTM connection and listens for messages
func (b *SlackBot) Start() {
	var botID string
	api := slack.New(b.Token)
	if len(b.SlackURL) != 0 {
		api = slack.New(b.Token, slack.OptionAPIURL(b.SlackURL))
		botID = b.BotID
	} else {
		authResp, err := api.AuthTest()
		if err != nil {
			logging.Logger.Fatal(err)
		}
		botID = authResp.UserID
	}

	RTM := api.NewRTM()
	go RTM.ManageConnection()

	for msg := range RTM.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			logging.Logger.Info("BotKube connected to Slack!")

		case *slack.MessageEvent:
			// Skip if message posted by BotKube
			if ev.User == botID {
				continue
			}
			sm := slackMessage{
				Event:       ev,
				BotID:       botID,
				RTM:         RTM,
				SlackClient: api,
			}
			sm.HandleMessage(b)

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.ConnectionErrorEvent:
			logging.Logger.Errorf("Slack connection error: %+v", ev.Error())

		case *slack.IncomingEventError:
			logging.Logger.Errorf("Slack incoming event error: %+v", ev.Error())

		case *slack.OutgoingErrorEvent:
			logging.Logger.Errorf("Slack outgoing event error: %+v", ev.Error())

		case *slack.UnmarshallingErrorEvent:
			logging.Logger.Errorf("Slack unmarshalling error: %+v", ev.Error())

		case *slack.RateLimitedError:
			logging.Logger.Errorf("Slack rate limiting error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return

		default:
		}
	}
}

func (sm *slackMessage) HandleMessage(b *SlackBot) {
	logging.Logger.Debugf("Slack incoming message: %+v", sm.Event)

	// Check if message posted in authenticated channel
	info, err := sm.SlackClient.GetConversationInfo(sm.Event.Channel, true)
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
	// Serve only if current channel is in config
	if b.ChannelName == sm.Event.Channel {
		sm.IsAuthChannel = true
	}

	// Trim the @BotKube prefix
	sm.Request = strings.TrimPrefix(sm.Event.Text, "<@"+sm.BotID+"> ")
	if len(sm.Request) == 0 {
		return
	}

	e := execute.NewDefaultExecutor(sm.Request, b.AllowKubectl, b.RestrictAccess, b.ClusterName, b.ChannelName, sm.IsAuthChannel)
	sm.Response = e.Execute()
	sm.Send()
}

func (sm slackMessage) Send() {
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
			logging.Logger.Error("Error in uploading file:", err)
		}
		return
	} else if len(sm.Response) == 0 {
		logging.Logger.Info("Invalid request. Dumping the response")
		return
	}

	if _, _, err := sm.RTM.PostMessage(sm.Event.Channel, slack.MsgOptionText("```"+sm.Response+"```", false), slack.MsgOptionAsUser(true)); err != nil {
		logging.Logger.Error("Error in sending message:", err)
	}
}
