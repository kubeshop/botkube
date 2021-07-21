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
	"net/url"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/mattermost/mattermost-server/model"
)

// mmChannelType to find Mattermost channel type
type mmChannelType string

const (
	mmChannelPrivate mmChannelType = "P"
	mmChannelPublic  mmChannelType = "O"
	mmChannelDM      mmChannelType = "D"
)

const (
	// WebSocketProtocol stores protocol initials for web socket
	WebSocketProtocol = "ws://"
	// WebSocketSecureProtocol stores protocol initials for web socket
	WebSocketSecureProtocol = "wss://"
)

// MMBot listens for user's message, execute commands and sends back the response
type MMBot struct {
	Token            string
	BotName          string
	TeamName         string
	ChannelName      string
	ClusterName      string
	AllowKubectl     bool
	RestrictAccess   bool
	ServerURL        string
	WebSocketURL     string
	WSClient         *model.WebSocketClient
	APIClient        *model.Client4
	DefaultNamespace string
}

// mattermostMessage contains message details to execute command and send back the result
type mattermostMessage struct {
	Event         *model.WebSocketEvent
	Response      string
	Request       string
	IsAuthChannel bool
	APIClient     *model.Client4
}

// NewMattermostBot returns new Bot object
func NewMattermostBot(c *config.Config) Bot {
	return &MMBot{
		ServerURL:        c.Communications.Mattermost.URL,
		BotName:          c.Communications.Mattermost.BotName,
		Token:            c.Communications.Mattermost.Token,
		TeamName:         c.Communications.Mattermost.Team,
		ChannelName:      c.Communications.Mattermost.Channel,
		ClusterName:      c.Settings.ClusterName,
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
	}
}

// Start establishes mattermost connection and listens for messages
func (b *MMBot) Start() {
	b.APIClient = model.NewAPIv4Client(b.ServerURL)
	b.APIClient.SetOAuthToken(b.Token)

	// Check if Mattermost URL is valid
	checkURL, err := url.Parse(b.ServerURL)
	if err != nil {
		log.Errorf("The Mattermost URL entered is incorrect. URL: %s. Error: %s", b.ServerURL, err.Error())
		return
	}

	// Create WebSocketClient and handle messages
	b.WebSocketURL = WebSocketProtocol + checkURL.Host
	if checkURL.Scheme == "https" {
		b.WebSocketURL = WebSocketSecureProtocol + checkURL.Host
	}

	// Check connection to Mattermost server
	err = b.checkServerConnection()
	if err != nil {
		log.Fatalf("There was a problem pinging the Mattermost server URL %s. %s", b.ServerURL, err.Error())
		return
	}

	go func() {
		// It is observed that Mattermost server closes connections unexpectedly after some time.
		// For now, we are adding retry logic to reconnect to the server
		// https://github.com/infracloudio/botkube/issues/201
		log.Info("BotKube connected to Mattermost!")
		for {
			var appErr *model.AppError
			b.WSClient, appErr = model.NewWebSocketClient4(b.WebSocketURL, b.APIClient.AuthToken)
			if appErr != nil {
				log.Errorf("Error creating WebSocket for Mattermost connectivity. %v", appErr)
				return
			}
			b.listen()
		}
	}()
	return
}

// Check incoming message and take action
func (mm *mattermostMessage) handleMessage(b MMBot) {
	post := model.PostFromJson(strings.NewReader(mm.Event.Data["post"].(string)))
	channelType := mmChannelType(mm.Event.Data["channel_type"].(string))
	if channelType == mmChannelPrivate || channelType == mmChannelPublic {
		// Message posted in a channel
		// Serve only if starts with mention
		if !strings.HasPrefix(post.Message, "@"+b.BotName+" ") {
			return
		}
	}

	// Check if message posted in authenticated channel
	if mm.Event.Broadcast.ChannelId == b.getChannel().Id {
		mm.IsAuthChannel = true
	}
	log.Debugf("Received mattermost event: %+v", mm.Event.Data)

	// Trim the @BotKube prefix if exists
	mm.Request = strings.TrimPrefix(post.Message, "@"+b.BotName+" ")

	e := execute.NewDefaultExecutor(mm.Request, b.AllowKubectl, b.RestrictAccess, b.DefaultNamespace,
		b.ClusterName, config.MattermostBot, b.ChannelName, mm.IsAuthChannel)
	mm.Response = e.Execute()
	mm.sendMessage()
}

// Send messages to Mattermost
func (mm mattermostMessage) sendMessage() {
	log.Debugf("Mattermost incoming Request: %s", mm.Request)
	log.Debugf("Mattermost Response: %s", mm.Response)
	post := &model.Post{}
	post.ChannelId = mm.Event.Broadcast.ChannelId

	if len(mm.Response) == 0 {
		log.Infof("Invalid request. Dumping the response. Request: %s", mm.Request)
		return
	}
	// Create file if message is too large
	if len(mm.Response) >= 3990 {
		res, resp := mm.APIClient.UploadFileAsRequestBody([]byte(mm.Response), mm.Event.Broadcast.ChannelId, mm.Request)
		if resp.Error != nil {
			log.Error("Error occurred while uploading file. Error: ", resp.Error)
		}
		post.FileIds = []string{string(res.FileInfos[0].Id)}
	} else {
		post.Message = formatCodeBlock(mm.Response)
	}

	// Create a post in the Channel
	if _, resp := mm.APIClient.CreatePost(post); resp.Error != nil {
		log.Error("Failed to send message. Error: ", resp.Error)
	}
}

// Check if Mattermost server is reachable
func (b MMBot) checkServerConnection() error {
	// Check api connection
	if _, resp := b.APIClient.GetOldClientConfig(""); resp.Error != nil {
		return resp.Error
	}

	// Get channel list
	_, resp := b.APIClient.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Check if team exists in Mattermost
func (b MMBot) getTeam() *model.Team {
	botTeam, resp := b.APIClient.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		log.Fatalf("There was a problem finding Mattermost team %s. %s", b.TeamName, resp.Error)
	}
	return botTeam
}

// Check if BotKube user exists in Mattermost
func (b MMBot) getUser() *model.User {
	users, resp := b.APIClient.AutocompleteUsersInTeam(b.getTeam().Id, b.BotName, 1, "")
	if resp.Error != nil {
		log.Fatalf("There was a problem finding Mattermost user %s. %s", b.BotName, resp.Error)
	}
	return users.Users[0]
}

// Create channel if not present and add BotKube user in channel
func (b MMBot) getChannel() *model.Channel {
	// Checking if channel exists
	botChannel, resp := b.APIClient.GetChannelByName(b.ChannelName, b.getTeam().Id, "")
	if resp.Error != nil {
		log.Fatalf("There was a problem finding Mattermost channel %s. %s", b.ChannelName, resp.Error)
	}

	// Adding BotKube user to channel
	b.APIClient.AddChannelMember(botChannel.Id, b.getUser().Id)
	return botChannel
}

func (b MMBot) listen() {
	b.WSClient.Listen()
	defer b.WSClient.Close()
	for {
		if b.WSClient.ListenError != nil {
			log.Debugf("Mattermost websocket listen error %s. Reconnecting...", b.WSClient.ListenError)
			return
		}

		event := <-b.WSClient.EventChannel
		if event == nil {
			continue
		}
		if event.Event == model.WEBSOCKET_EVENT_POSTED {
			post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))

			// Skip if message posted by BotKube or doesn't start with mention
			if post.UserId == b.getUser().Id {
				continue
			}
			mm := mattermostMessage{
				Event:         event,
				IsAuthChannel: false,
				APIClient:     b.APIClient,
			}
			mm.handleMessage(b)
		}
	}
}
