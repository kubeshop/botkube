// Copyright (c) 2021 InfraCloud Technologies
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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/core/tools"
	"github.com/larksuite/oapi-sdk-go/event"
	eventhttpserver "github.com/larksuite/oapi-sdk-go/event/http/native"
)

const (
	//Event lark event
	Event = "event"
	//ChatType lark chat type
	ChatType = "chat_type"
	//Text lark chat message
	Text = "text_without_at_bot"
	//OpenChatID lark chat id
	OpenChatID = "open_chat_id"
	//ChatID lark chat id
	ChatID = "chat_id"
	//OpenID lark user id
	OpenID = "open_id"
	//Users lark users
	Users = "users"
	//UserID lark user id
	UserID = "user_id"
	//Message eventType When sending a message to a chat group
	Message = "message"
	//AddBot eventType When a bot is added to a chat group
	AddBot = "add_bot"
	//P2pChatCreate eventType When a session is first created with the bot
	P2pChatCreate = "p2p_chat_create"
	//AddUserToChat eventType When a user is added to a chat group
	AddUserToChat = "add_user_to_chat"
)

// LarkBot listens for user's message, execute commands and sends back the response
type LarkBot struct {
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	DefaultNamespace string
	Port             int
	MessagePath      string
	LarkClient       *utils.LarkClient
}

// NewLarkBot returns new Bot object
func NewLarkBot(c *config.Config) Bot {
	larkConf := c.Communications.Lark
	appSettings := core.NewInternalAppSettings(core.SetAppCredentials(larkConf.AppID, larkConf.AppSecret),
		core.SetAppEventKey(larkConf.VerificationToken, larkConf.EncryptKey))
	conf := core.NewConfig(core.Domain(larkConf.Endpoint), appSettings, core.SetLoggerLevel(core.LoggerLevelError))
	return &LarkBot{
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
		Port:             c.Communications.Lark.Port,
		MessagePath:      c.Communications.Lark.MessagePath,
		LarkClient:       utils.NewLarkClient(conf),
	}
}

// Execute commands sent by users
func (l *LarkBot) Execute(e map[string]interface{}) {
	event, ok := e[Event].(map[string]interface{})
	if !ok {
		log.Error("Missing expected event object in the request")
		return
	}

	chatType, ok := event[ChatType].(string)
	if !ok {
		log.Error("Missing expected chatType object in the request")
		return
	}

	text, ok := event[Text].(string)
	if !ok {
		log.Error("Missing expected text object in the request")
		return
	}

	executor := execute.NewDefaultExecutor(text, l.AllowKubectl, l.RestrictAccess, l.DefaultNamespace,
		l.ClusterName, config.LarkBot, "", true)
	response := executor.Execute()

	if chatType == "group" {
		chatID, ok := event[OpenChatID].(string)
		if !ok {
			log.Error("Missing expected chatID object in the request")
			return
		}
		l.LarkClient.SendTextMessage(ChatID, chatID, response)
		return
	}
	openID, ok := event[OpenID].(string)
	if !ok {
		log.Error("Missing expected openID object in the request")
		return
	}
	l.LarkClient.SendTextMessage(OpenID, openID, response)
}

// Start starts the lark server and listens for lark messages
func (l *LarkBot) Start() {
	// See: https://open.larksuite.com/document/ukTMukTMukTM/ukjNxYjL5YTM24SO2EjN
	// message
	larkConf := l.LarkClient.Conf
	event.SetTypeCallback(larkConf, Message, func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.Execute(e)
		return nil
	})

	// add_bot
	event.SetTypeCallback(larkConf, AddBot, func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})

	// p2p_chat_create
	event.SetTypeCallback(larkConf, P2pChatCreate, func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})

	// add_user_to_chat
	event.SetTypeCallback(larkConf, AddUserToChat, func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})

	eventhttpserver.Register(l.MessagePath, larkConf)
	log.Infof("Started lark server on port %d", l.Port)
	log.Errorf("Error in lark server. %v", http.ListenAndServe(fmt.Sprintf(":%d", l.Port), nil))
}

// SayHello send welcome message to new added users
func (l *LarkBot) SayHello(e map[string]interface{}) error {
	event, ok := e[Event].(map[string]interface{})
	if !ok {
		return larkError(Event)
	}
	users, ok := event[Users].([]interface{})
	if !ok {
		user := event[Users].(interface{})
		users = append(users, user)
	}

	var messages []string
	if users != nil {
		for _, user := range users {
			openID, ok := user.(map[string]interface{})[OpenID].(string)
			if !ok {
				log.Error("Missing expected openID object in the request")
			}
			username := user.(map[string]interface{})[UserID].(string)
			if !ok {
				log.Error("Missing expected username object in the request")
			}
			messages = append(messages, fmt.Sprintf("<at user_id=\"%s\">%s</at>", openID, username))
		}
	}
	messages = append(messages, "Hello from botkube~ Play with me by at botkube <commands>")
	chatID, ok := event[ChatID].(string)
	if !ok {
		return larkError("chatID")
	}
	return l.LarkClient.SendTextMessage(ChatID, chatID, strings.Join(messages, " "))
}

func larkError(str string) error {
	return errors.New("Missing expected " + str + " object in the request")
}
