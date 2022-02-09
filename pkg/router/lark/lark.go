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

package lark

import (
	"context"
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/router/config"
	"github.com/infracloudio/botkube/pkg/utils"

	"github.com/larksuite/oapi-sdk-go/core"
	larkconfig "github.com/larksuite/oapi-sdk-go/core/config"
	"github.com/larksuite/oapi-sdk-go/core/tools"
	"github.com/larksuite/oapi-sdk-go/event"
	im "github.com/larksuite/oapi-sdk-go/service/im/v1"
)

// RouterBot listens for user's message, execute commands and sends back the response
type RouterBot struct {
	Conf        *larkconfig.Config
	Port        int
	MessagePath string
	Service     *im.Service
}

// NewLarkBot returns new LarkBot object
func NewLarkBot(c *config.RouterConfig) *RouterBot {
	larkConfig := c.Communications.Lark
	appSettings := core.NewInternalAppSettings(
		core.SetAppCredentials(larkConfig.AppID, larkConfig.AppSecret),
		core.SetAppEventKey(larkConfig.VerificationToken, larkConfig.EncryptKey))
	conf := core.NewConfig(core.Domain(larkConfig.Endpoint), appSettings, core.SetLoggerLevel(core.LoggerLevelInfo))
	imService := im.NewService(conf)
	return &RouterBot{
		Conf:        conf,
		Port:        larkConfig.Port,
		MessagePath: larkConfig.MessagePath,
		Service:     imService,
	}
}

// Start starts the lark server and listens for lark messages
func (l *RouterBot) Start() {
	// add_bot
	event.SetTypeCallback(l.Conf, "add_bot", func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})

	// p2p_chat_create
	event.SetTypeCallback(l.Conf, "p2p_chat_create", func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})

	// See: https://open.f.mioffice.cn/document/ukTMukTMukTM/ukjNxYjL5YTM24SO2EjN
	// add_user_to_chat
	event.SetTypeCallback(l.Conf, "add_user_to_chat", func(ctx *core.Context, e map[string]interface{}) error {
		log.Infof(ctx.GetRequestID())
		log.Infof(tools.Prettify(e))
		go l.SayHello(e)
		return nil
	})
}

//SayHello handles the user's first group entry
func (l *RouterBot) SayHello(e map[string]interface{}) {
	event, ok := e["event"].(map[string]interface{})
	if !ok {
		log.Error("Missing expected event object in the request")
		return
	}
	users, ok := event["users"].([]interface{})
	if !ok {
		user, ok := event["users"].(interface{})
		if !ok {
			log.Error("Missing expected user object in the request")
			return
		}
		users = append(users, user)
	}
	var messages []string
	if users != nil {
		for _, user := range users {
			openID, ok := user.(map[string]interface{})["open_id"].(string)
			if !ok {
				log.Error("Missing expected openID object in the request")
			}
			username := user.(map[string]interface{})["user_id"].(string)
			if !ok {
				log.Error("Missing expected username object in the request")
			}
			messages = append(messages, fmt.Sprintf("<at user_id=\"%s\">%s</at>", openID, username))
		}
	}
	messages = append(messages, "Hello from botkube~ Play with me by at botkube <commands>")
	l.SendMessage("chat_id", event["chat_id"].(string), strings.Join(messages, " "))
}

// SendMessage sends message to slack channel
// See: https://open.f.mioffice.cn/document/uAjLw4CM/ukTMukTMukTM/im-v1/message/create_json
func (l *RouterBot) SendMessage(receiveType, receiveID, msg string) error {
	coreCtx := core.WrapContext(context.Background())
	content := utils.LarkMessage(msg)

	reqCall := l.Service.Messages.Create(coreCtx, &im.MessageCreateReqBody{
		ReceiveId: receiveID,
		Content:   content,
		MsgType:   "text",
	})
	reqCall.SetReceiveIdType(receiveType)
	message, err := reqCall.Do()
	if err != nil {
		log.Errorf("Error in sending lark message: %s error: %+v", msg, err)
		return err
	}

	log.Infof("Message successfully sent to channel %s with message: %s", receiveID, message.Body.Content)
	return nil
}
