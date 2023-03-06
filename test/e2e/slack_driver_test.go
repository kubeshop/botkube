//go:build integration

package e2e

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
)

type SlackChannel struct {
	*slack.Channel
}

func (s *SlackChannel) ID() string {
	return s.Channel.ID
}
func (s *SlackChannel) Name() string {
	return s.Channel.Name
}
func (s *SlackChannel) Identifier() string {
	return s.Channel.Name
}

type slackTester struct {
	cli           *slack.Client
	cfg           SlackConfig
	botUserID     string
	testerUserID  string
	channel       Channel
	secondChannel Channel
	mdFormatter   interactive.MDFormatter
}

func newSlackDriver(slackCfg SlackConfig) (BotDriver, error) {
	slackCli := slack.New(slackCfg.TesterAppToken)
	_, err := slackCli.AuthTest()
	if err != nil {
		return nil, err
	}
	mdFormatter := interactive.NewMDFormatter(interactive.NewlineFormatter, func(msg string) string {
		return fmt.Sprintf("*%s*", msg)
	})
	return &slackTester{cli: slackCli, cfg: slackCfg, mdFormatter: mdFormatter}, nil
}

func (s *slackTester) InitUsers(t *testing.T) {
	t.Helper()
	s.botUserID = s.findUserID(t, s.cfg.BotName)
	assert.NotEmpty(t, s.botUserID, "could not find slack botUserID with name: %s", s.cfg.BotName)

	s.testerUserID = s.findUserID(t, s.cfg.TesterName)
	assert.NotEmpty(t, s.testerUserID, "could not find slack testerUserID with name: %s", s.cfg.TesterName)
}

func (s *slackTester) InitChannels(t *testing.T) []func() {
	channel, cleanupChannelFn := s.createChannel(t, "first")
	s.channel = &SlackChannel{Channel: channel}

	secondChannel, cleanupSecondChannelFn := s.createChannel(t, "second")
	s.secondChannel = &SlackChannel{Channel: secondChannel}

	return []func(){
		func() { cleanupChannelFn(t) },
		func() { cleanupSecondChannelFn(t) },
	}
}

func (s *slackTester) Type() DriverType {
	return SlackBot
}

func (s *slackTester) BotName() string {
	return fmt.Sprintf("<@%s>", s.BotUserID())
}

func (s *slackTester) BotUserID() string {
	return s.botUserID
}

func (s *slackTester) TesterUserID() string {
	return s.testerUserID
}

func (s *slackTester) Channel() Channel {
	return s.channel
}

func (s *slackTester) SecondChannel() Channel {
	return s.secondChannel
}

func (s *slackTester) PostInitialMessage(t *testing.T, channelName string) {
	t.Helper()
	t.Log("Posting welcome message...")

	var additionalContextMsg string
	if s.cfg.AdditionalContextMessage != "" {
		additionalContextMsg = fmt.Sprintf("%s\n", s.cfg.AdditionalContextMessage)
	}
	message := fmt.Sprintf("Hello!\n%s%s", additionalContextMsg, welcomeText)
	_, _, err := s.cli.PostMessage(channelName, slack.MsgOptionText(message, false))
	require.NoError(t, err)
}

func (s *slackTester) PostMessageToBot(t *testing.T, channel, command string) {
	message := fmt.Sprintf("<@%s> %s", s.cfg.BotName, command)
	_, _, err := s.cli.PostMessage(channel, slack.MsgOptionText(message, false))
	require.NoError(t, err)
}

func (s *slackTester) InviteBotToChannel(t *testing.T, channelID string) {
	t.Logf("Inviting bot with ID %q to the channel with ID %q", s.botUserID, channelID)
	_, err := s.cli.InviteUsersToConversation(channelID, s.botUserID)
	require.NoError(t, err)
}

func (s *slackTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) (bool, int, string) {
		msg = s.trimNewLine(msg)
		if !strings.EqualFold(expectedMsg, msg) {
			count := countMatchBlock(expectedMsg, msg)
			msgDiff := diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *slackTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		return strings.Contains(s.trimNewLine(msg), expectedMsgSubstring), 0, ""
	})
}

func (s *slackTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		msg = s.trimNewLine(msg)
		msg = formatx.RemoveHyperlinks(msg) // normalize the message URLs
		if msg != expectedMsg {
			count := countMatchBlock(expectedMsg, msg)
			msgDiff := diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *slackTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	var fetchedMessages []slack.Message
	var lastErr error
	var diffMessage string
	var highestCommonBlockCount int
	if limitMessages == 1 {
		highestCommonBlockCount = -1 // a single message is fetched, always print diff
	}

	err := wait.Poll(pollInterval, s.cfg.MessageWaitTimeout, func() (done bool, err error) {
		historyRes, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channelID, Limit: limitMessages,
		})
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = historyRes.Messages
		for _, msg := range historyRes.Messages {
			if msg.User != userID {
				continue
			}
			equal, commonCount, diffStr := assertFn(msg.Text)
			if !equal {
				// different message; update the diff if it's more similar than the previous one or initial value
				if commonCount > highestCommonBlockCount {
					highestCommonBlockCount = commonCount
					diffMessage = diffStr
				}
				continue
			}

			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = fmt.Errorf("message assertion function returned false%s", diffMessage)
	}
	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *slackTester) WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	return s.WaitForMessagePosted(userID, channelID, limitMessages, assertFn)
}

func (s *slackTester) WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error {
	var fetchedMessages []slack.Message
	var lastErr error
	err := wait.Poll(pollInterval, s.cfg.MessageWaitTimeout, func() (done bool, err error) {
		historyRes, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channelID, Limit: 1,
		})
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = historyRes.Messages
		for _, msg := range historyRes.Messages {
			if msg.User != userID {
				continue
			}

			if len(msg.Files) != 1 {
				return false, nil
			}

			upload := msg.Files[0]
			if !assertFn(upload.Title, upload.Mimetype) {
				// different message
				return false, nil
			}

			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = errors.New("message assertion function returned false")
	}
	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *slackTester) WaitForMessagePostedWithAttachment(userID, channelID string, limitMessages int, assertFn ExpAttachmentInput) error {
	renderer := bot.NewSlackRenderer()

	var expTime time.Time
	if !assertFn.Message.Timestamp.IsZero() {
		expTime = assertFn.Message.Timestamp
		assertFn.Message.Timestamp = time.Time{}
	}

	// we don't support the attachment anymore, so content is available as normal message
	return s.WaitForMessagePosted(userID, channelID, limitMessages, func(content string) (bool, int, string) {
		// for now, we use old slack, so we send messages as a markdown
		expMsg := renderer.MessageToMarkdown(interactive.CoreMessage{
			Message: assertFn.Message,
		})

		if !expTime.IsZero() {
			body, timestamp := s.cutLastLine(content)
			content = body // update content, so timestamp doesn't impact static content assertion

			gotEventTime, err := dateparse.ParseAny(timestamp)
			if err != nil {
				return false, 0, err.Error()
			}

			if err = timeWithinDuration(expTime, gotEventTime, time.Minute); err != nil {
				return false, 0, err.Error()
			}
		}

		expMsg = replaceEmojiWithTags(expMsg)
		if !strings.EqualFold(expMsg, content) {
			count := countMatchBlock(expMsg, content)
			msgDiff := diff(expMsg, content)
			return false, count, msgDiff
		}

		return true, 0, ""
	})
}

// TODO: This contains an implementation for socket mode slack apps. Once needed, you can see the already implemented
// functions here https://github.com/kubeshop/botkube/blob/abfeb95fa5f84ceb9b25a30159cdc3d17e130711/test/e2e/slack_driver_test.go#L289
func (s *slackTester) WaitForInteractiveMessagePostedRecentlyEqual(userID, channelID string, msg interactive.CoreMessage) error {
	renderedMsg := interactive.RenderMessage(s.mdFormatter, msg)
	return s.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) (bool, int, string) {
		// Slack encloses URLs with `<` and `>`, since we need to remove them before assertion
		msg = strings.NewReplacer("<https", "https", ">\n", "\n").Replace(msg)
		if !strings.EqualFold(renderedMsg, msg) {
			count := countMatchBlock(renderedMsg, msg)
			msgDiff := diff(renderedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *slackTester) WaitForLastInteractiveMessagePostedEqual(userID, channelID string, msg interactive.CoreMessage) error {
	renderedMsg := interactive.RenderMessage(s.mdFormatter, msg)
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		msg = strings.NewReplacer("<https", "https", ">\n", "\n").Replace(msg)
		if !strings.EqualFold(renderedMsg, msg) {
			count := countMatchBlock(renderedMsg, msg)
			msgDiff := diff(renderedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *slackTester) findUserID(t *testing.T, name string) string {
	t.Log("Getting users...")
	res, err := s.cli.GetUsers()
	require.NoError(t, err)

	t.Logf("Finding user ID by name %q...", name)
	for _, u := range res {
		if u.Name != name {
			continue
		}
		return u.ID
	}

	return ""
}

func (s *slackTester) createChannel(t *testing.T, prefix string) (*slack.Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s-%s", channelNamePrefix, prefix, randomID.String())

	t.Logf("Creating channel %q...", channelName)
	// > There’s no limit to how many unique channels you can have in Slack — go ahead, create as many as you’d like!
	// Sure, thanks Slack!
	// Source: https://slack.com/help/articles/201402297-Create-a-channel
	channel, err := s.cli.CreateConversation(slack.CreateConversationParams{ChannelName: channelName, IsPrivate: false})
	require.NoError(t, err)

	t.Logf("Channel %q (ID: %q) created", channelName, channel.ID)

	cleanupFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Archiving channel %q...", channel.Name)
		// We cannot delete channel: https://stackoverflow.com/questions/46807744/delete-channel-in-slack-api
		err = s.cli.ArchiveConversation(channel.ID)
		assert.NoError(t, err)
	}

	return channel, cleanupFn
}

func (s *slackTester) trimNewLine(msg string) string {
	// There is always a `\n` on Slack messages due to Markdown formatting.
	// That should be replaced for RTM
	return strings.TrimSuffix(msg, "\n")
}

func (s *slackTester) cutLastLine(in string) (before string, after string) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", ""
	}
	lastNewLine := strings.LastIndexAny(in, "\n")
	if lastNewLine == -1 {
		return in, ""
	}

	return in[:lastNewLine], in[lastNewLine+1:]
}

var emojiSlackMapping = map[string]string{
	"🟢":  ":large_green_circle:",
	"⚠️": ":warning:",
	"ℹ️": ":information_source:",
	"❗":  ":exclamation:",
}

func replaceEmojiWithTags(content string) string {
	for emoji, tag := range emojiSlackMapping {
		content = strings.ReplaceAll(content, emoji, tag)
	}
	return content
}
