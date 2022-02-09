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

package utils

import (
	"context"
	"encoding/json"

	"github.com/infracloudio/botkube/pkg/log"
	"github.com/larksuite/oapi-sdk-go/core"
	larkconfig "github.com/larksuite/oapi-sdk-go/core/config"
	im "github.com/larksuite/oapi-sdk-go/service/im/v1"
)

// LarkClient the client to communication with lark open platform
type LarkClient struct {
	Conf    *larkconfig.Config
	Service *im.Service
}

// TextMessage lark text message
type TextMessage struct {
	Text string `json:"text,omitempty" validate:"omitempty"`
}

// NewLarkClient returns new Lark client
func NewLarkClient(conf *larkconfig.Config) *LarkClient {
	imService := im.NewService(conf)
	return &LarkClient{Conf: conf, Service: imService}
}

func (lark *LarkClient) marshalTextMessage(message string) (string, error) {
	content := &TextMessage{Text: message}
	data, err := json.Marshal(content)
	if err != nil {
		log.Errorf("Error in marshal message: %s error: %+v", content, err)
		return "", err
	}
	return string(data), nil
}

// SendTextMessage sending text message to lark group
// See: https://open.larksuite.com/document/ukTMukTMukTM/uUjNz4SN2MjL1YzM
func (lark *LarkClient) SendTextMessage(receiveType, receiveID, msg string) error {
	content, err := lark.marshalTextMessage(msg)
	if err != nil {
		log.Errorf("Error in sending marshal text message: %s error: %+v", msg, err)
		return err
	}
	coreCtx := core.WrapContext(context.Background())
	reqCall := lark.Service.Messages.Create(coreCtx, &im.MessageCreateReqBody{
		ReceiveId: receiveID,
		Content:   content,
		MsgType:   "text",
	})
	reqCall.SetReceiveIdType(receiveType)
	ret, err := reqCall.Do()
	if err != nil {
		log.Errorf("Error in sending lark message: %s error: %+v", msg, err)
		return err
	}
	log.Debugf("Message successfully sent to channel: %s with message: %+v", receiveID, ret)
	return nil
}

//MessageContent lark message body
type MessageContent struct {
	Text string `json:"text,omitempty" validate:"omitempty"`
}

//LarkMessage formatting lark message body
func LarkMessage(message string) string {
	content := &MessageContent{Text: message}
	data, err := json.Marshal(content)
	if err != nil {
		log.Errorf("Error in marshal message: %s error: %+v", content, err)
		return ""
	}
	return string(data)
}
