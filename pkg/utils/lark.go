package utils

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larksuite/oapi-sdk-go/core"
	larkconfig "github.com/larksuite/oapi-sdk-go/core/config"
	im "github.com/larksuite/oapi-sdk-go/service/im/v1"
	"github.com/sirupsen/logrus"
)

// LarkClient the client to communication with lark open platform
type LarkClient struct {
	log     logrus.FieldLogger
	Conf    *larkconfig.Config
	Service *im.Service
}

// TextMessage lark text message
type TextMessage struct {
	Text string `json:"text,omitempty" validate:"omitempty"`
}

// NewLarkClient returns new Lark client
func NewLarkClient(log logrus.FieldLogger, conf *larkconfig.Config) *LarkClient {
	imService := im.NewService(conf)
	return &LarkClient{log: log, Conf: conf, Service: imService}
}

func (lark *LarkClient) marshalTextMessage(message string) (string, error) {
	content := &TextMessage{Text: message}
	data, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("Error in marshal message: %s error: %s", content, err.Error())
	}
	return string(data), nil
}

// SendTextMessage sending text message to lark group
// See: https://open.larksuite.com/document/ukTMukTMukTM/uUjNz4SN2MjL1YzM
func (lark *LarkClient) SendTextMessage(ctx context.Context, receiveType, receiveID, msg string) error {
	content, err := lark.marshalTextMessage(msg)
	if err != nil {
		return fmt.Errorf("while sending text message %q: %w", msg, err)
	}
	coreCtx := core.WrapContext(ctx)
	reqCall := lark.Service.Messages.Create(coreCtx, &im.MessageCreateReqBody{
		ReceiveId: receiveID,
		Content:   content,
		MsgType:   "text",
	})
	reqCall.SetReceiveIdType(receiveType)
	ret, err := reqCall.Do()
	if err != nil {
		return fmt.Errorf("Error in sending lark message: %s error: %s", msg, err.Error())
	}
	lark.log.Debugf("Message successfully sent to channel: %s with message: %+v", receiveID, ret)
	return nil
}

//GetLoggerLevel Unified Log Levels
func GetLoggerLevel(loggerLevel logrus.Level) core.LoggerLevel {
	switch int(loggerLevel) {
	case 0, 1, 2:
		return core.LoggerLevelError
	case 3:
		return core.LoggerLevelWarn
	case 4:
		return core.LoggerLevelInfo
	case 5, 6:
		return core.LoggerLevelDebug
	default:
		return core.LoggerLevelError
	}
}
