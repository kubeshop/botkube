package bot

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/logging"
	"github.com/mattermost/mattermost-server/model"
)

var client *model.Client4

const (
	// BotName stores Botkube details
	BotName = "botkube"
	// WebSocketProtocol stores protocol initials for web socket
	WebSocketProtocol = "ws://"
	// WebSocketSecureProtocol stores protocol initials for web socket
	WebSocketSecureProtocol = "wss://"
)

// mmBot listens for user's message, execute commands and sends back the response
type mmBot struct {
	ServerURL    string
	Token        string
	TeamName     string
	ChannelName  string
	ClusterName  string
	AllowKubectl bool
}

// mattermostMessage contains message details to execute command and send back the result
type mattermostMessage struct {
	Event         *model.WebSocketEvent
	Response      string
	Request       string
	IsAuthChannel bool
}

// NewMattermostBot returns new Bot object
func NewMattermostBot() Bot {
	c, err := config.New()
	if err != nil {
		logging.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	return &mmBot{
		ServerURL:    c.Communications.Mattermost.URL,
		Token:        c.Communications.Mattermost.Token,
		TeamName:     c.Communications.Mattermost.Team,
		ChannelName:  c.Communications.Mattermost.Channel,
		ClusterName:  c.Settings.ClusterName,
		AllowKubectl: c.Settings.AllowKubectl,
	}
}

// Start establishes mattermost connection and listens for messages
func (b *mmBot) Start() {
	client = model.NewAPIv4Client(b.ServerURL)
	client.SetOAuthToken(b.Token)

	// Check if Mattermost URL is valid
	checkURL, err := url.Parse(b.ServerURL)
	if err != nil {
		logging.Logger.Error("The Mattermost URL entered is incorrect. URL: ", b.ServerURL, "\nError: ", err)
		return
	}

	// Check connection to Mattermost server
	err = checkServerConnection()
	if err != nil {
		logging.Logger.Error("There was a problem pinging the Mattermost server URL: ", b.ServerURL, "\nError: ", err)
		return
	}

	// Create WebSocketClient and handle messages
	webSocketURL := WebSocketProtocol + checkURL.Host
	if checkURL.Scheme == "https" {
		webSocketURL = WebSocketSecureProtocol + checkURL.Host
	}
	var modelError *model.AppError
	webSocketClient, modelError := model.NewWebSocketClient4(webSocketURL, client.AuthToken)
	if modelError != nil {
		logging.Logger.Error("Error creating WebSocket for Mattermost connectivity. \nError: ", modelError)
		return
	}

	webSocketClient.Listen()
	go func() {
		for {
			event := <-webSocketClient.EventChannel
			if event.Event != model.WEBSOCKET_EVENT_POSTED {
				continue
			}
			post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))

			// Skip if message posted by BotKube or doesn't start with mention
			if post.UserId == b.getUser().Id || !(strings.HasPrefix(post.Message, "@"+BotName+" ")) {
				continue
			}
			mm := mattermostMessage{
				Event:         event,
				IsAuthChannel: false,
			}
			mm.handleMessage(b)
		}
	}()
	return
}

// Check incomming message and take action
func (mm *mattermostMessage) handleMessage(b *mmBot) {
	// Check if message posted in authenticated channel
	if mm.Event.Broadcast.ChannelId == b.getChannel().Id {
		mm.IsAuthChannel = true
	}

	post := model.PostFromJson(strings.NewReader(mm.Event.Data["post"].(string)))
	// Trim the @BotKube prefix
	mm.Request = strings.TrimPrefix(post.Message, "@"+BotName+" ")

	e := execute.NewDefaultExecutor(mm.Request, b.AllowKubectl, b.ClusterName, b.ChannelName, mm.IsAuthChannel)
	mm.Response = e.Execute()
	mm.sendMessage()
}

// Send messages to Mattermost
func (mm mattermostMessage) sendMessage() {
	post := &model.Post{}
	post.ChannelId = mm.Event.Broadcast.ChannelId
	// Create file if message is too large
	if len(mm.Response) >= 3990 {
		res, resp := client.UploadFileAsRequestBody([]byte(mm.Response), mm.Event.Broadcast.ChannelId, mm.Request)
		if resp.Error != nil {
			logging.Logger.Error("Error occured while uploading file. Error: ", resp.Error)
		}
		post.FileIds = []string{string(res.FileInfos[0].Id)}
	} else if len(mm.Response) == 0 {
		logging.Logger.Info("Invalid request. Dumping the response")
		return
	} else {
		post.Message = "```\n" + mm.Response + "\n```"
	}

	// Create a post in the Channel
	if _, resp := client.CreatePost(post); resp.Error != nil {
		logging.Logger.Error("Failed to send message. Error: ", resp.Error)
	}
}

// Check if Mattermost server is reachable
func checkServerConnection() error {
	if _, resp := client.GetOldClientConfig(""); resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Check if team exists in Mattermost
func (b *mmBot) getTeam() *model.Team {
	botTeam, resp := client.GetTeamByName(b.TeamName, "")
	if resp.Error != nil {
		logging.Logger.Fatal("There was a problem finding Mattermost team ", b.TeamName, "\nError: ", resp.Error)
	}
	return botTeam
}

// Check if botkube user exists in Mattermost
func (b *mmBot) getUser() *model.User {
	users, resp := client.AutocompleteUsersInTeam(b.getTeam().Id, BotName, "")
	if resp.Error != nil {
		logging.Logger.Fatal("There was a problem finding Mattermost user ", BotName, "\nError: ", resp.Error)
	}
	return users.Users[0]
}

// Create channel if not present and add botkube user in channel
func (b *mmBot) getChannel() *model.Channel {
	// Checking if channel exists
	botChannel, resp := client.GetChannelByName(b.ChannelName, b.getTeam().Id, "")
	if resp.Error != nil {
		logging.Logger.Fatal("There was a problem finding Mattermost channel ", b.ChannelName, "\nError: ", resp.Error)
	}

	// Adding Botkube user to channel
	client.AddChannelMember(botChannel.Id, b.getUser().Id)
	return botChannel
}
