package commplatform

import (
	"strings"
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
	Channel() Channel
	SecondChannel() Channel
	ThirdChannel() Channel
	BotName() string
	BotUserID() string
	TesterUserID() string
	MDFormatter() interactive.MDFormatter
	WaitForInteractiveMessagePostedRecentlyEqual(userID string, channelID string, message interactive.CoreMessage) error
	WaitForLastInteractiveMessagePostedEqual(userID string, channelID string, message interactive.CoreMessage) error
	WaitForLastInteractiveMessagePostedEqualWithCustomRender(userID, channelID string, renderedMsg string) error
	SetTimeout(timeout time.Duration)
	Timeout() time.Duration
	ReplaceBotNamePlaceholder(msg *interactive.CoreMessage, clusterName string)
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
)

// AssertContains checks if message contains expected message
func AssertContains(expectedMessage string) MessageAssertion {
	return func(msg string) (bool, int, string) {
		return strings.Contains(msg, expectedMessage), 0, ""
	}
}

// AssertEquals checks if message is equal to expected message
func AssertEquals(expectedMessage string) MessageAssertion {
	return func(msg string) (bool, int, string) {
		return msg == expectedMessage, 0, ""
	}
}
