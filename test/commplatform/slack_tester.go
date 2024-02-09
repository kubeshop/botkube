package commplatform

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"testing"
	"time"

	"botkube.io/botube/test/diff"
	"github.com/araddon/dateparse"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/ptr"
)

var _ BotDriver = (*SlackTester)(nil)

const (
	slackInteractiveElementsMsgSuffix = ", with interactive elements"
)

var slackLinks = regexp.MustCompile(`<(?P<val>https://[^>]*)>`)

type SlackConfig struct {
	BotName                  string        `envconfig:"default=botkube"`
	CloudBotName             string        `envconfig:"default=botkubedev"`
	CloudBasedTestEnabled    bool          `envconfig:"default=true"`
	TesterName               string        `envconfig:"default=tester"`
	AdditionalContextMessage string        `envconfig:"optional"`
	TesterAppToken           string        `envconfig:"optional"`
	TesterBotToken           string        `envconfig:"optional"`
	CloudTesterAppToken      string        `envconfig:"optional"`
	CloudTesterName          string        `envconfig:"default=tester2"`
	RecentMessagesLimit      int           `envconfig:"default=6"`
	MessageWaitTimeout       time.Duration `envconfig:"default=50s"`
}

type SlackChannel struct {
	*slack.Channel
}

type SlackMessageAssertion func(content slack.Message) (bool, int, string)

func (s *SlackChannel) ID() string {
	return s.Channel.ID
}
func (s *SlackChannel) Name() string {
	return s.Channel.Name
}
func (s *SlackChannel) Identifier() string {
	return s.Channel.Name
}

type SlackTester struct {
	cli                        *slack.Client
	cfg                        SlackConfig
	botUserID                  string
	testerUserID               string
	firstChannel               Channel
	secondChannel              Channel
	thirdChannel               Channel
	mdFormatter                interactive.MDFormatter
	configProviderApiKey       string
	recentlyPostedMsgTS        map[string]string
	channelsByName             map[string]Channel
	restoreRecentlyPostedMsgTS func()
}

func (s *SlackTester) ReplaceBotNamePlaceholder(msg *interactive.CoreMessage, clusterName string) {
	msg.ReplaceBotNamePlaceholder(s.BotName(), api.BotNameWithClusterName(clusterName))
}

func NewSlackTester(slackCfg SlackConfig, apiKey *string) (*SlackTester, error) {
	var token string
	if slackCfg.TesterAppToken == "" && slackCfg.TesterBotToken == "" && slackCfg.CloudTesterAppToken == "" {
		return nil, errors.New("slack tester tokens are not set")
	}

	if slackCfg.TesterAppToken != "" {
		token = slackCfg.TesterAppToken
	}
	if slackCfg.TesterBotToken != "" {
		token = slackCfg.TesterBotToken
	}
	if slackCfg.CloudBasedTestEnabled && slackCfg.CloudTesterAppToken != "" {
		token = slackCfg.CloudTesterAppToken
	}

	slackCli := slack.New(token)
	_, err := slackCli.AuthTest()
	if err != nil {
		return nil, err
	}
	mdFormatter := interactive.NewMDFormatter(interactive.NewlineFormatter, func(msg string) string {
		return fmt.Sprintf("*%s*", msg)
	})
	return &SlackTester{
		cli:                        slackCli,
		cfg:                        slackCfg,
		mdFormatter:                mdFormatter,
		configProviderApiKey:       ptr.ToValue(apiKey),
		recentlyPostedMsgTS:        make(map[string]string),
		channelsByName:             make(map[string]Channel),
		restoreRecentlyPostedMsgTS: func() {},
	}, nil
}

func (s *SlackTester) InitUsers(t *testing.T) {
	t.Helper()
	botName := s.cfg.BotUsername()
	s.botUserID = s.findUserID(t, botName)
	assert.NotEmpty(t, s.botUserID, "could not find slack botUserID with name: %s", botName)

	s.testerUserID = s.findUserID(t, s.cfg.CloudTesterName)
	assert.NotEmpty(t, s.testerUserID, "could not find slack testerUserID with name: %s", s.cfg.CloudTesterName)
}

func (s *SlackTester) InitChannels(t *testing.T) []func() {
	channel, cleanupChannelFn := s.CreateChannel(t, "first")
	s.firstChannel = channel

	secondChannel, cleanupSecondChannelFn := s.CreateChannel(t, "second")
	s.secondChannel = secondChannel

	thirdChannel, cleanupThirdChannelFn := s.CreateChannel(t, "rbac")
	s.thirdChannel = thirdChannel

	return []func(){
		func() { cleanupChannelFn(t) },
		func() { cleanupSecondChannelFn(t) },
		func() { cleanupThirdChannelFn(t) },
	}
}

func (s *SlackTester) Type() DriverType {
	return SlackBot
}

func (s *SlackTester) BotName() string {
	return fmt.Sprintf("<@%s>", s.BotUserID())
}

func (s *SlackTester) BotUserID() string {
	return s.botUserID
}

func (s *SlackTester) TesterUserID() string {
	return s.testerUserID
}

func (s *SlackTester) FirstChannel() Channel {
	return s.firstChannel
}

func (s *SlackTester) SecondChannel() Channel {
	return s.secondChannel
}

func (s *SlackTester) ThirdChannel() Channel {
	return s.thirdChannel
}

func (s *SlackTester) PostInitialMessage(t *testing.T, channelName string) {
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

func (s *SlackTester) PostMessageToBot(t *testing.T, channelName, command string) {
	message := fmt.Sprintf("<@%s> %s", s.cfg.BotUsername(), command)
	_, ts, err := s.cli.PostMessage(channelName, slack.MsgOptionText(message, false))
	id := s.channelsByName[channelName].ID()
	s.recentlyPostedMsgTS[id] = ts
	require.NoError(t, err)
}

func (s *SlackTester) InviteBotToChannel(t *testing.T, channelID string) {
	t.Logf("Inviting bot with ID %q to the channel with ID %q", s.botUserID, channelID)
	_, err := s.cli.InviteUsersToConversation(channelID, s.botUserID)
	require.NoError(t, err)
}

func (s *SlackTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, s.cfg.RecentMessagesLimit, s.AssertEquals(expectedMsg))
}

func (s *SlackTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		return strings.Contains(msg, expectedMsgSubstring), 0, ""
	})
}

func (s *SlackTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, s.AssertEquals(expectedMsg))
}

// AssertEquals checks if message is equal to expected message
func (s *SlackTester) AssertEquals(expectedMsg string) MessageAssertion {
	return func(msg string) (bool, int, string) {
		msg = formatx.RemoveHyperlinks(msg) // normalize the message URLs
		msg = strings.NewReplacer("<https", "https", ">\n", "\n").Replace(msg)
		msg = strings.ReplaceAll(msg, slackInteractiveElementsMsgSuffix, "") // remove interactive elements suffix
		if !strings.EqualFold(expectedMsg, msg) {
			count := diff.CountMatchBlock(expectedMsg, msg)
			msgDiff := diff.Diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	}
}

func (s *SlackTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	defer s.restoreMsgTsIfNeeded()

	var fetchedMessages []slack.Message
	var lastErr error
	var diffMessage string
	var highestCommonBlockCount int
	if limitMessages == 1 {
		highestCommonBlockCount = -1 // a single message is fetched, always print diff
	}

	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
		fetchedMessages, err = s.getMessages(channelID, limitMessages)
		if err != nil {
			lastErr = err
			return false, nil
		}

		for _, msg := range fetchedMessages {
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
		lastErr = fmt.Errorf("message assertion function returned false; fetched messages: %s\ndiff:\n%s", structDumper.Sdump(fetchedMessages), diffMessage)
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *SlackTester) getMessages(channelID string, limitMessages int) ([]slack.Message, error) {
	if ts := s.recentlyPostedMsgTS[channelID]; ts != "" {
		replies, _, _, err := s.cli.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: ts,
			Limit:     limitMessages,
		})
		return replies, err
	}

	history, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: channelID, Limit: limitMessages,
	})
	return history.Messages, err
}

func (s *SlackTester) WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	defer s.restoreMsgTsIfNeeded()
	var fetchedMessages []slack.Message
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(_ context.Context) (done bool, err error) {
		fetchedMessages, err = s.getMessages(channelID, limitMessages)
		if err != nil {
			lastErr = err
			return false, nil
		}

		for _, msg := range fetchedMessages {
			if msg.User != userID {
				continue
			}

			if len(msg.Blocks.BlockSet) == 0 {
				continue
			}

			ok, _, _ := assertFn(sPrintBlocks(s.normalizeSlackBlockSet(msg.Blocks)))

			if !ok {
				continue
			}
			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = fmt.Errorf("message assertion function returned false; fetched messages: %s", structDumper.Sdump(fetchedMessages))
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *SlackTester) WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error {
	defer s.restoreMsgTsIfNeeded()
	var fetchedMessages []slack.Message
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
		fetchedMessages, err := s.getMessages(channelID, 2) // Fetching 2 messages, where the first one is the file itself and the second is the filter input.
		if err != nil {
			lastErr = err
			return false, nil
		}

		for _, msg := range fetchedMessages {
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
		lastErr = fmt.Errorf("message assertion function returned false; fetched messages: %s", structDumper.Sdump(fetchedMessages))
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *SlackTester) WaitForMessagePostedWithAttachment(userID, channelID string, limitMessages int, assertFn ExpAttachmentInput) error {
	defer s.restoreMsgTsIfNeeded()
	renderer := bot.NewSlackRenderer()

	var expTime time.Time
	if !assertFn.Message.Timestamp.IsZero() {
		expTime = assertFn.Message.Timestamp
		assertFn.Message.Timestamp = time.Time{}
	}

	// we don't support the attachment anymore, so content is available as normal message
	return s.WaitForMessagePosted(userID, channelID, limitMessages, func(content string) (bool, int, string) {
		// for now, we use old slack, so we send messages as a markdown
		expMsg := normalizeAttachmentContent(renderer.MessageToMarkdown(interactive.CoreMessage{
			Message: assertFn.Message,
		}))
		if !expTime.IsZero() {
			body, timestamp := s.trimAttachmentTimestamp(content)
			content = normalizeAttachmentContent(body)
			gotEventTime, err := dateparse.ParseAny(timestamp)
			if err != nil {
				return false, 0, err.Error()
			}

			if err = diff.TimeWithinDuration(expTime, gotEventTime, time.Minute); err != nil {
				return false, 0, err.Error()
			}
		}

		expMsg = replaceEmojiWithTags(expMsg)
		if !strings.EqualFold(expMsg, content) {
			count := diff.CountMatchBlock(expMsg, content)
			msgDiff := diff.Diff(expMsg, content)
			return false, count, msgDiff
		}

		return true, 0, ""
	})
}

func (s *SlackTester) WaitForInteractiveMessagePostedRecentlyEqual(userID, channelID string, msg interactive.CoreMessage) error {
	printedBlocks := sPrintBlocks(bot.NewSlackRenderer().RenderAsSlackBlocks(msg))
	return s.WaitForInteractiveMessagePosted(userID, channelID, s.cfg.RecentMessagesLimit, s.AssertEquals(printedBlocks))
}

func sPrintBlocks(blocks []slack.Block) string {
	var builder strings.Builder

	for _, block := range blocks {
		switch block.BlockType() {
		case slack.MBTSection:
			section := block.(*slack.SectionBlock)
			builder.WriteString("::::")
			builder.WriteString(fmt.Sprintf("section: %s", section.Text.Text))
		case slack.MBTDivider:
			builder.WriteString("::::")
			builder.WriteString("divider")
		case slack.MBTAction:
			action := block.(*slack.ActionBlock)
			builder.WriteString("::::")
			for _, element := range action.Elements.ElementSet {
				switch element.ElementType() {
				case slack.METButton:
					button := element.(*slack.ButtonBlockElement)
					builder.WriteString(fmt.Sprintf("action::button: %s <> %s <> %s",
						button.Text.Text,
						button.Value,
						button.ActionID,
					))
				}
			}
		}
	}
	builder.WriteString("::::")
	return builder.String()
}

func (s *SlackTester) WaitForLastInteractiveMessagePostedEqual(userID, channelID string, msg interactive.CoreMessage) error {
	printedBlocks := sPrintBlocks(bot.NewSlackRenderer().RenderAsSlackBlocks(msg))
	return s.WaitForInteractiveMessagePosted(userID, channelID, 1, s.AssertEquals(printedBlocks))
}


func (s *SlackTester) SetTimeout(timeout time.Duration) {
	s.cfg.MessageWaitTimeout = timeout
}

func (s *SlackTester) Timeout() time.Duration {
	return s.cfg.MessageWaitTimeout
}

func (s *SlackTester) normalizeSlackBlockSet(got slack.Blocks) []slack.Block {
	for idx, item := range got.BlockSet {
		switch item.BlockType() {
		case slack.MBTSection:
			item := item.(*slack.SectionBlock)
			item.BlockID = "" // it's generated by SDK, so we don't compare it.
			if item.Text != nil {
				item.Text.Text = removeSlackLinksIndicators(item.Text.Text)
			}

			got.BlockSet[idx] = item
		case slack.MBTDivider:
			item := item.(*slack.DividerBlock)
			item.BlockID = "" // it's generated by SDK, so we don't compare it.
			got.BlockSet[idx] = item
		case slack.MBTAction:
			item := item.(*slack.ActionBlock)
			item.BlockID = "" // it's generated by SDK, so we don't compare it.
			got.BlockSet[idx] = item
		}
	}
	return got.BlockSet
}

func (s *SlackTester) findUserID(t *testing.T, name string) string {
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

func (s *SlackTester) CreateChannel(t *testing.T, prefix string) (Channel, func(t *testing.T)) {
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
	out := &SlackChannel{Channel: channel}

	s.channelsByName[out.Name()] = out
	return out, cleanupFn
}

func (s *SlackConfig) BotUsername() string {
	if s.CloudBasedTestEnabled {
		return s.CloudBotName
	}
	return s.BotName
}

func TrimSlackMsgTrailingLine(msg string) string {
	// There is always a `\n` on Slack messages due to Markdown formatting.
	// That should be replaced for RTM
	return strings.TrimSuffix(msg, "\n")
}

func normalizeAttachmentContent(msg string) string {
	msg = strings.ReplaceAll(msg, " • ", "")
	msg = strings.ReplaceAll(msg, "• ", "")
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "   *", " *")
	msg = strings.ReplaceAll(msg, "*Fields* ", "")
	msg = strings.ReplaceAll(msg, "*: ", ":* ")
	msg = strings.TrimSuffix(msg, "  ")
	return msg
}

func (s *SlackTester) trimAttachmentTimestamp(in string) (string, string) {
	msgParts := strings.Split(in, " <!date^")
	ts := ""
	if len(msgParts) > 1 {
		ts = strings.Split(msgParts[1], "^")[0]
	}
	return msgParts[0], ts
}

// OnChannel removes expectation that messages should be posted in the thread of recently sent message.
// After first assertion it restores expectation.
func (s *SlackTester) OnChannel() BotDriver {
	old := make(map[string]string)
	maps.Copy(old, s.recentlyPostedMsgTS)
	s.recentlyPostedMsgTS = map[string]string{}
	s.restoreRecentlyPostedMsgTS = func() {
		s.recentlyPostedMsgTS = old
	}
	return s
}

func (s *SlackTester) restoreMsgTsIfNeeded() {
	if s.restoreRecentlyPostedMsgTS != nil {
		s.restoreRecentlyPostedMsgTS()
	}
	s.restoreRecentlyPostedMsgTS = nil
}

var emojiSlackMapping = map[string]string{
	"🟢": ":large_green_circle:",
	"💡": ":bulb:",
	"❗": ":exclamation:",
}

func replaceEmojiWithTags(content string) string {
	for emoji, tag := range emojiSlackMapping {
		content = strings.ReplaceAll(content, emoji, tag)
	}
	return content
}

func removeSlackLinksIndicators(content string) string {
	tpl := "$val"

	return slackLinks.ReplaceAllStringFunc(content, func(s string) string {
		var result []byte
		result = slackLinks.ExpandString(result, tpl, s, slackLinks.FindSubmatchIndex([]byte(s)))
		return string(result)
	})
}
