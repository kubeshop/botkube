// Copyright (c) 2019 InfraCloud Technologies
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
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/slack-go/slack"
)

// SlackBot listens for user's message, execute commands and sends back the response
type SlackBot struct {
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
	Event         *slack.MessageEvent
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	RTM           *slack.RTM
	SlackClient   *slack.Client
}

// NewSlackBot returns new Bot object
func NewSlackBot(c *config.Config) Bot {
	return &SlackBot{
		Token:            c.Communications.Slack.Token,
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		ChannelName:      c.Communications.Slack.Channel,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
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
			log.Fatal(err)
		}
		botID = authResp.UserID
	}

	RTM := api.NewRTM()
	go RTM.ManageConnection()

	for msg := range RTM.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			log.Info("BotKube connected to Slack!")

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
			log.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.ConnectionErrorEvent:
			log.Errorf("Slack connection error: %+v", ev.Error())

		case *slack.IncomingEventError:
			log.Errorf("Slack incoming event error: %+v", ev.Error())

		case *slack.OutgoingErrorEvent:
			log.Errorf("Slack outgoing event error: %+v", ev.Error())

		case *slack.UnmarshallingErrorEvent:
			log.Errorf("Slack unmarshalling error: %+v", b.stripUnmarshallingErrEventDetails(ev.Error()))

		case *slack.RateLimitedError:
			log.Errorf("Slack rate limiting error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			log.Error("Invalid Credentials")
			return

		default:
		}
	}
}

// Temporary workaround until the PR is merged: https://github.com/slack-go/slack/pull/1067
// There are just two cases when the full event details are concatenated with the error message: https://github.com/slack-go/slack/blob/v0.10.3/websocket_managed_conn.go#L474-L492
// Also, we need keep the JSON unmarshal error intact: https://github.com/slack-go/slack/blob/v0.10.3/websocket_managed_conn.go#L405
func (b *SlackBot) stripUnmarshallingErrEventDetails(errMessage string) string {
	const errMsgSeparator = ": "
	const prefix = "RTM Error"

	if !strings.HasPrefix(errMessage, prefix) {
		return errMessage
	}

	// has prefix, so let's split the message into parts
	msgParts := strings.Split(errMessage, errMsgSeparator)
	if len(msgParts) < 3 {
		// impossible with current version of the dependency
		return errMessage
	}

	return strings.Join(msgParts[:2], errMsgSeparator)
}

func (sm *slackMessage) HandleMessage(b *SlackBot) {
	// Check if message posted in authenticated channel
	info, err := sm.SlackClient.GetConversationInfo(sm.Event.Channel, true)
	if err == nil {
		if info.IsChannel || info.IsPrivate {
			// Message posted in a channel
			// Serve only if starts with mention
			if !strings.HasPrefix(sm.Event.Text, "<@"+sm.BotID+">") {
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
	sm.Request = strings.TrimPrefix(sm.Event.Text, "<@"+sm.BotID+">")

	e := execute.NewDefaultExecutor(sm.Request, b.AllowKubectl, b.RestrictAccess, b.DefaultNamespace,
		b.ClusterName, config.SlackBot, b.ChannelName, sm.IsAuthChannel)
	sm.Response = e.Execute()
	sm.Send()
}

func (sm *slackMessage) Send() {
	log.Debugf("Slack incoming Request: %s", sm.Request)
	log.Debugf("Slack Response: %s", sm.Response)
	if len(sm.Response) == 0 {
		log.Infof("Invalid request. Dumping the response. Request: %s", sm.Request)
		return
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
			log.Error("Error in uploading file:", err)
		}
		return
	}

	var options = []slack.MsgOption{slack.MsgOptionText(formatCodeBlock(sm.Response), false), slack.MsgOptionAsUser(true)}

	//if the message is from thread then add an option to return the response to the thread
	if sm.Event.ThreadTimestamp != "" {
		options = append(options, slack.MsgOptionTS(sm.Event.ThreadTimestamp))
	}

	if _, _, err := sm.RTM.PostMessage(sm.Event.Channel, options...); err != nil {
		log.Error("Error in sending message:", err)
	}
}
