// Copyright (c) 2022 InfraCloud Technologies
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
	"fmt"
	"net/http"
	"strings"

	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/event"
	eventhttpserver "github.com/larksuite/oapi-sdk-go/event/http/native"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	//larkEvent lark event
	larkEvent = "event"
	//larkChatType lark chat type
	larkChatType = "chat_type"
	//larkGroup lark chat type
	larkGroup = "group"
	//larkText lark chat message
	larkText = "text_without_at_bot"
	//larkOpenChatID lark chat id
	larkOpenChatID = "open_chat_id"
	//larkChatID lark chat id
	larkChatID = "chat_id"
	//larkOpenID lark user id
	larkOpenID = "open_id"
	//larkUsers lark users
	larkUsers = "users"
	//larkUserID lark user id
	larkUserID = "user_id"
	//larkMessage eventType When sending a message to a chat group
	larkMessage = "message"
	//larkAddBot eventType When a bot is added to a chat group
	larkAddBot = "add_bot"
	//larkP2pChatCreate eventType When a session is first created with the bot
	larkP2pChatCreate = "p2p_chat_create"
	//larkAddUserToChat eventType When a user is added to a chat group
	larkAddUserToChat = "add_user_to_chat"
	//larkStartMsg lark start message
	larkStartMsg = "Hello from BotKube. Visit botkube.io to know more."
	//larkAtUser lark at user message
	larkAtUser = "<at user_id=\"'%s'\">'%s'</at>"
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
	conf := core.NewConfig(core.Domain(larkConf.Endpoint), appSettings, core.SetLoggerLevel(utils.GetLoggerLevel()))
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
func (l *LarkBot) Execute(e map[string]interface{}) error {
	event, err := accessAndTypeCastToMap(larkEvent, e)
	if err != nil {
		log.Errorf("Lark: AccessAndTypeCastToMap: %s", err.Error())
		return err
	}

	chatType, err := accessAndTypeCastToString(larkChatType, event)
	if err != nil {
		log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
		return err
	}

	text, err := accessAndTypeCastToString(larkText, event)
	if err != nil {
		log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
		return err
	}

	executor := execute.NewDefaultExecutor(text, l.AllowKubectl, l.RestrictAccess, l.DefaultNamespace,
		l.ClusterName, config.LarkBot, "", true)
	response := executor.Execute()

	if chatType == larkGroup {
		chatID, err := accessAndTypeCastToString(larkOpenChatID, event)
		if err != nil {
			log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
			return err
		}

		err = l.LarkClient.SendTextMessage(larkChatID, chatID, response)
		if err != nil {
			log.Errorf("Lark: SendTextMessage: %s", err.Error())
			return err
		}
	}
	openID, err := accessAndTypeCastToString(larkOpenID, event)
	if err != nil {
		log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
		return err
	}
	err = l.LarkClient.SendTextMessage(larkOpenID, openID, response)
	if err != nil {
		log.Errorf("Lark: SendTextMessage: %s", err.Error())
		return err
	}
	return nil
}

// Start starts the lark server and listens for lark messages
func (l *LarkBot) Start() error {
	// See: https://open.larksuite.com/document/ukTMukTMukTM/ukjNxYjL5YTM24SO2EjN
	// message
	larkConf := l.LarkClient.Conf
	event.SetTypeCallback(larkConf, larkMessage, func(ctx *core.Context, e map[string]interface{}) error {
		return l.Execute(e)
	})

	// add_bot
	event.SetTypeCallback(larkConf, larkAddBot, func(ctx *core.Context, e map[string]interface{}) error {
		return l.SayHello(e)
	})

	// p2p_chat_create
	event.SetTypeCallback(larkConf, larkP2pChatCreate, func(ctx *core.Context, e map[string]interface{}) error {
		return l.SayHello(e)
	})

	// add_user_to_chat
	event.SetTypeCallback(larkConf, larkAddUserToChat, func(ctx *core.Context, e map[string]interface{}) error {
		return l.SayHello(e)
	})

	eventhttpserver.Register(l.MessagePath, larkConf)
	log.Infof("Started lark server on port %d", l.Port)
	log.Errorf("Error in lark server. %v", http.ListenAndServe(fmt.Sprintf(":%d", l.Port), nil))
	return nil
}

// SayHello send welcome message to new added users
func (l *LarkBot) SayHello(e map[string]interface{}) error {
	event, err := accessAndTypeCastToMap(larkEvent, e)
	if err != nil {
		return err
	}

	larkUserList, ok := event[larkUsers]
	if !ok {
		return fmt.Errorf("Lark user %s not found", larkUsers)
	}
	users, ok := larkUserList.([]interface{})
	if !ok {
		user := event[larkUsers]
		users = append(users, user)
	}

	var messages []string
	if users == nil {
		return fmt.Errorf("Lark user is nil")
	}
	for _, user := range users {
		userMap, ok := user.(map[string]interface{})
		if !ok {
			log.Error("Invalid user format. Failed to convert into user into map[string]interface{}")
			continue
		}
		openID, err := accessAndTypeCastToString(larkOpenID, userMap)
		if err != nil {
			log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
			continue
		}
		username, err := accessAndTypeCastToString(larkUserID, userMap)
		if err != nil {
			log.Errorf("Lark: AccessAndTypeCastToString: %s", err.Error())
			continue
		}

		messages = append(messages, fmt.Sprintf(larkAtUser, openID, username))
	}

	messages = append(messages, larkStartMsg)

	chatID, err := accessAndTypeCastToString(larkChatID, event)
	if err != nil {
		return err
	}
	return l.LarkClient.SendTextMessage(larkChatID, chatID, strings.Join(messages, " "))
}
