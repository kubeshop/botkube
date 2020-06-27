package domain

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ITBot Listens to commands and message. Also listens to callBacks.
// This is designed to be immutable
type ITBot struct {
	token            string
	allowKubectl     bool
	restrictAccess   bool
	clusterName      string
	groupID          int64
	defaultNamespace string
	debug            bool
}

//WithConfig Used to create a new bot with a given config
func (bot ITBot) WithConfig(
	token string,
	allowKubectl bool,
	restrictAccess bool,
	clusterName string,
	groupID int64,
	defaultNamespace string,
	debug bool) ITBot {
	bot.token = token
	bot.allowKubectl = allowKubectl
	bot.restrictAccess = restrictAccess
	bot.clusterName = clusterName
	bot.groupID = groupID
	bot.defaultNamespace = defaultNamespace
	bot.debug = debug
	return bot
}

//Token telegram token
func (bot ITBot) Token() string {
	return bot.token
}

// AllowKubectl previlage
func (bot ITBot) AllowKubectl() bool {
	return bot.allowKubectl
}

// RestrictAccess previlage
func (bot ITBot) RestrictAccess() bool {
	return bot.restrictAccess
}

// ClusterName info
func (bot ITBot) ClusterName() string {
	return bot.clusterName
}

// GroupID of the telegram group. Usually is is a negetive value
func (bot ITBot) GroupID() int64 {
	return bot.groupID
}

//DefaultNamespace configuration
func (bot ITBot) DefaultNamespace() string {
	return bot.defaultNamespace
}

func (bot ITBot) Debug() bool {
	return bot.debug
}

//ITMsg is an immutable message used to hold intermediate results of the pipeline
type ITMsg struct {
	request     tgbotapi.Update
	responseTxt string
	response    tgbotapi.MessageConfig
	command     string
}

func (tMsg ITMsg) WithRequest(request tgbotapi.Update) ITMsg {
	tMsg.request = request
	return tMsg
}
func (tMsg ITMsg) WithResponseTxt(responseTxt string) ITMsg {
	tMsg.responseTxt = responseTxt
	return tMsg
}
func (tMsg ITMsg) WithResponse(response tgbotapi.MessageConfig) ITMsg {
	tMsg.response = response
	return tMsg
}
func (tMsg ITMsg) WithCommand(command string) ITMsg {
	tMsg.command = command
	return tMsg
}
func (tMsg ITMsg) IsLongResponse() bool {
	if len(tMsg.responseTxt) >= 3990 {
		return true
	}
	return false
}

func (tMsg ITMsg) HasKeyBoard() bool {
	if tMsg.response.ReplyMarkup != nil {
		return true
	}
	return false
}

func (tMsg ITMsg) Request() tgbotapi.Update {
	return tMsg.request
}
func (tMsg ITMsg) ResponseTxt() string {
	return tMsg.responseTxt
}
func (tMsg ITMsg) Response() tgbotapi.MessageConfig {
	return tMsg.response
}
func (tMsg ITMsg) Command() string {
	return tMsg.command
}
