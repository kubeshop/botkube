package commplatform

import (
	"testing"
	"time"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

const (
	channelNamePrefix = "test"
	welcomeText       = "Let the tests begin 🤞"

	pollInterval = time.Second
)

type Channel interface {
	ID() string
	Name() string
	Identifier() string
}

type BotDriver interface {
	Type() DriverType
	InitUsers(t *testing.T)

	CreateChannel(t *testing.T, prefix string) (Channel, func(t *testing.T))
	InitChannels(t *testing.T) []func()
	PostInitialMessage(t *testing.T, channel string)
	PostMessageToBot(t *testing.T, channel, command string)
	InviteBotToChannel(t *testing.T, channel string)
	WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error
	WaitForLastMessageContains(userID, channel, expectedMsgSubstring string) error
	WaitForLastMessageEqual(userID, channel, expectedMsg string) error
	WaitForMessagePosted(userID, channel string, limitMessages int, assertFn MessageAssertion) error
	WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error
	WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error
	WaitForMessagePostedWithAttachment(userID, channel string, limitMessages int, expInput ExpAttachmentInput) error
	FirstChannel() Channel
	SecondChannel() Channel
	ThirdChannel() Channel
	BotName() string
	BotUserID() string
	TesterUserID() string
	WaitForInteractiveMessagePostedRecentlyEqual(userID string, channelID string, message interactive.CoreMessage) error
	WaitForLastInteractiveMessagePostedEqual(userID string, channelID string, message interactive.CoreMessage) error
	SetTimeout(timeout time.Duration)
	Timeout() time.Duration
	ReplaceBotNamePlaceholder(msg *interactive.CoreMessage, clusterName string)
	AssertEquals(expectedMessage string) MessageAssertion
	// OnChannel sets the expectation that the message should be posted in the channel. This is necessary when Bots
	// by default expect a given message to be posted in the thread of the recently sent message.
	// For example, in the context of source notification, we need to alter that default behavior
	// and expect the message on the channel instead.
	OnChannel() BotDriver
}

type MessageAssertion func(content string) (bool, int, string)

type FileUploadAssertion func(title, mimetype string) bool

type ExpAttachmentInput struct {
	Message               api.Message
	AllowedTimestampDelta time.Duration
}

// DriverType to instrument
type DriverType string

const (
	SlackBot   DriverType = "cloudSlack"
	DiscordBot DriverType = "discord"
	TeamsBot   DriverType = "teams"
)

func (d DriverType) IsCloud() bool {
	switch d {
	case SlackBot, TeamsBot:
		return true
	default:
		return false
	}
}

// AssertEquals checks if message is equal to expected message
func (d DriverType) AssertEquals(expectedMessage string) MessageAssertion {
	return func(msg string) (bool, int, string) {
		return msg == expectedMessage, 0, ""
	}
}
