package notify

import (
	"context"

	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	//ChatID  lark chat group id
	ChatID = "chat_id"
)

// Lark contains LarkClient for communication with lark and receiver group name to send notification to
type Lark struct {
	log           logrus.FieldLogger
	LarkClient    *utils.LarkClient
	ReceiverGroup string
}

// NewLark returns new Lark object
func NewLark(log logrus.FieldLogger, loggerLevel logrus.Level, c config.CommunicationsConfig) *Lark {
	appSettings := core.NewInternalAppSettings(core.SetAppCredentials(c.Lark.AppID, c.Lark.AppSecret),
		core.SetAppEventKey(c.Lark.VerificationToken, c.Lark.EncryptKey))
	conf := core.NewConfig(core.Domain(c.Lark.Endpoint), appSettings, core.SetLoggerLevel(utils.GetLoggerLevel(loggerLevel)))
	return &Lark{
		log:           log,
		LarkClient:    utils.NewLarkClient(log, conf),
		ReceiverGroup: c.Lark.ChatGroup,
	}
}

// SendEvent sends event notification to lark chart group
func (l *Lark) SendEvent(ctx context.Context, event events.Event) error {
	l.log.Debugf(">> Sending to lark: %+v", event)
	return l.LarkClient.SendTextMessage(ctx, ChatID, l.ReceiverGroup, FormatShortMessage(event))
}

// SendMessage sends message to lark chart group
func (l *Lark) SendMessage(ctx context.Context, msg string) error {
	l.log.Debugf(">> Sending to lark: %+v", msg)
	return l.LarkClient.SendTextMessage(ctx, ChatID, l.ReceiverGroup, msg)
}
