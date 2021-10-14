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

package notify

import (
	"github.com/larksuite/oapi-sdk-go/core"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	//ChatID  lark chat group id
	ChatID = "chat_id"
)

// Lark contains LarkClient for communication with lark and receiver group name to send notification to
type Lark struct {
	LarkClient    *utils.LarkClient
	ReceiverGroup string
}

// NewLark returns new Lark object
func NewLark(c config.CommunicationsConfig) Notifier {
	appSettings := core.NewInternalAppSettings(core.SetAppCredentials(c.Lark.AppID, c.Lark.AppSecret),
		core.SetAppEventKey(c.Lark.VerificationToken, c.Lark.EncryptKey))
	conf := core.NewConfig(core.Domain(c.Lark.Endpoint), appSettings, core.SetLoggerLevel(utils.GetLoggerLevel()))
	return &Lark{LarkClient: utils.NewLarkClient(conf),
		ReceiverGroup: c.Lark.ChatGroup}
}

// SendEvent sends event notification to lark chart group
func (l *Lark) SendEvent(event events.Event) error {
	log.Debugf(">> Sending to lark: %+v", event)
	return l.LarkClient.SendTextMessage(ChatID, l.ReceiverGroup, FormatShortMessage(event))
}

// SendMessage sends message to lark chart group
func (l *Lark) SendMessage(msg string) error {
	log.Debugf(">> Sending to lark: %+v", msg)
	return l.LarkClient.SendTextMessage(ChatID, l.ReceiverGroup, msg)
}
