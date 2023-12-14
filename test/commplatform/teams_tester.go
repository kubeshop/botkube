package commplatform

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/kubeshop/botkube/pkg/api/cloudteams"
	"github.com/kubeshop/botkube/test/msteamsx"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/test/diff"
)

const (
	platformMessageWorkersCount = 10
	platformMessageChannelSize  = 100
)

type TeamsConfig struct {
	BotName                  string        `envconfig:"default=botkube"`
	CloudBotName             string        `envconfig:"default=botkubedev"`
	CloudBasedTestEnabled    bool          `envconfig:"default=true"`
	TesterName               string        `envconfig:"default=tester"`
	AdditionalContextMessage string        `envconfig:"optional"`
	CloudTesterAppToken      string        `envconfig:"optional"`
	CloudTesterName          string        `envconfig:"default=tester2"`
	RecentMessagesLimit      int           `envconfig:"default=6"`
	MessageWaitTimeout       time.Duration `envconfig:"default=50s"`
	AppID                    string
	AppPassword              string
	TenantID                 string
	TeamID                   string
	AADGroupID               string
}

type TeamsChannel struct {
	id   string
	name string
}

func (s *TeamsChannel) ID() string {
	return s.id
}
func (s *TeamsChannel) Name() string {
	return s.name
}
func (s *TeamsChannel) Identifier() string {
	return s.id
}

type TeamsTester struct {
	cli                  *msteamsx.Client
	cfg                  TeamsConfig
	botUserID            string
	testerUserID         string
	channel              Channel
	secondChannel        Channel
	thirdChannel         Channel
	mdFormatter          interactive.MDFormatter
	configProviderApiKey string
	agentActivityMessage chan *pb.AgentActivity
}

func (s *TeamsTester) ReplaceBotNamePlaceholder(msg *interactive.CoreMessage, clusterName string) {
	msg.ReplaceBotNamePlaceholder(s.BotName(), api.BotNameWithClusterName(clusterName))
}

func NewTeamsTester(teamsCfg TeamsConfig, apiKey *string) (BotDriver, error) {
	teamsCli, err := msteamsx.New(teamsCfg.AppID, teamsCfg.AppPassword, teamsCfg.TenantID)
	if err != nil {
		return nil, err
	}
	mdFormatter := interactive.NewMDFormatter(interactive.NewlineFormatter, func(msg string) string {
		return fmt.Sprintf("*%s*", msg)
	})
	return &TeamsTester{
		cli:                  teamsCli,
		cfg:                  teamsCfg,
		mdFormatter:          mdFormatter,
		configProviderApiKey: ptr.ToValue(apiKey),
		agentActivityMessage: make(chan *pb.AgentActivity, platformMessageChannelSize),
	}, nil
}

func (s *TeamsTester) InitUsers(t *testing.T) {
	t.Helper()
	t.Log("No need to init users for Teams, skipping...")
}

func (s *TeamsTester) InitChannels(t *testing.T) []func() {
	channel, cleanupChannelFn := s.CreateChannel(t, "first")
	s.channel = channel

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

func (s *TeamsTester) Type() DriverType {
	return TeamsBot
}

func (s *TeamsTester) BotName() string {
	return fmt.Sprintf("<@%s>", s.BotUserID())
}

func (s *TeamsTester) BotUserID() string {
	return s.cfg.TeamID
}

func (s *TeamsTester) TesterUserID() string {
	return s.testerUserID
}

func (s *TeamsTester) Channel() Channel {
	return s.channel
}

func (s *TeamsTester) SecondChannel() Channel {
	return s.secondChannel
}

func (s *TeamsTester) ThirdChannel() Channel {
	return s.thirdChannel
}

func (s *TeamsTester) MDFormatter() interactive.MDFormatter {
	return s.mdFormatter
}

func (s *TeamsTester) PostInitialMessage(t *testing.T, channelName string) {
	t.Helper()
	t.Log("Posting welcome message...")

	var additionalContextMsg string
	if s.cfg.AdditionalContextMessage != "" {
		additionalContextMsg = fmt.Sprintf("%s\n", s.cfg.AdditionalContextMessage)
	}
	message := fmt.Sprintf("Hello!\n%s%s", additionalContextMsg, welcomeText)
	err := s.cli.SendMessage(context.Background(), channelName, message)
	require.NoError(t, err)
}

func (s *TeamsTester) PostMessageToBot(t *testing.T, channel, command string) {
	message := fmt.Sprintf("<@%s> %s", s.cfg.BotUsername(), command)
	err := s.cli.SendMessage(context.Background(), channel, message)
	require.NoError(t, err)
}

func (s *TeamsTester) InviteBotToChannel(t *testing.T, channelID string) {
	t.Logf("No need to invite bot for channel %q since bot is added in Team level...", channelID)
}

func (s *TeamsTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, s.cfg.RecentMessagesLimit, func(msg string) (bool, int, string) {
		if !strings.EqualFold(expectedMsg, msg) {
			count := diff.CountMatchBlock(expectedMsg, msg)
			msgDiff := diff.Diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		return strings.Contains(msg, expectedMsgSubstring), 0, ""
	})
}

func (s *TeamsTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		msg = formatx.RemoveHyperlinks(msg)                                  // normalize the message URLs
		msg = strings.ReplaceAll(msg, slackInteractiveElementsMsgSuffix, "") // remove interactive elements suffix
		if msg != expectedMsg {
			count := diff.CountMatchBlock(expectedMsg, msg)
			msgDiff := diff.Diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	var fetchedMessages []slack.Message
	var lastErr error
	var diffMessage string
	/*var highestCommonBlockCount int
	if limitMessages == 1 {
		highestCommonBlockCount = -1 // a single message is fetched, always print diff
	}*/

	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
		/*historyRes, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
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
		}*/

		return false, nil
	})
	if lastErr == nil {
		lastErr = fmt.Errorf("message assertion function returned false%s", diffMessage)
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *TeamsTester) WaitForInteractiveMessagePosted(teamID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	var fetchedMessages []msteamsx.MsTeamsMessage
	var lastErr error
	// SA1019 suggested `PollWithContextTimeout` does not exist
	// nolint:staticcheck
	err := wait.Poll(pollInterval, s.cfg.MessageWaitTimeout, func() (done bool, err error) {
		historyRes, err := s.cli.GetMessages(context.Background(), teamID, channelID, limitMessages)
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = historyRes
		for _, msg := range fetchedMessages {
			log.Println(msg.Raw.GetFrom().GetApplication().GetDisplayName())
			log.Println(s.BotName())
			log.Println(msg.Rendered)
			if !strings.EqualFold(ptr.ToValue(msg.Raw.GetFrom().GetApplication().GetDisplayName()), s.BotName()) {
				continue
			}
			ok, _, _ := assertFn(msg.Rendered)

			if !ok {
				continue
			}

			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = errors.New("message assertion function returned false")
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *TeamsTester) WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error {
	var fetchedMessages []slack.Message
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
		/*	historyRes, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
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
			}*/

		return false, nil
	})
	if lastErr == nil {
		lastErr = errors.New("message assertion function returned false")
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *TeamsTester) WaitForMessagePostedWithAttachment(userID, channelID string, limitMessages int, assertFn ExpAttachmentInput) error {
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

func (s *TeamsTester) WaitForInteractiveMessagePostedRecentlyEqual(teamID, channelID string, msg interactive.CoreMessage) error {
	msgMd := bot.NewTeamsRenderer().MessageToMarkdown(msg)
	return s.WaitForInteractiveMessagePosted(teamID, channelID, s.cfg.RecentMessagesLimit, func(msg string) (bool, int, string) {
		if !strings.EqualFold(msg, msgMd) {
			count := diff.CountMatchBlock(msgMd, msg)
			msgDiff := diff.Diff(msgMd, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForLastInteractiveMessagePostedEqual(userID, channelID string, msg interactive.CoreMessage) error {
	printedBlocks := sPrintBlocks(bot.NewSlackRenderer().RenderAsSlackBlocks(msg))
	return s.WaitForInteractiveMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		if !strings.EqualFold(printedBlocks, msg) {
			count := diff.CountMatchBlock(printedBlocks, msg)
			msgDiff := diff.Diff(printedBlocks, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForLastInteractiveMessagePostedEqualWithCustomRender(userID, channelID string, renderedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		msg = strings.NewReplacer("<https", "https", ">\n", "\n").Replace(msg)
		if !strings.EqualFold(renderedMsg, msg) {
			count := diff.CountMatchBlock(renderedMsg, msg)
			msgDiff := diff.Diff(renderedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) SetTimeout(timeout time.Duration) {
	s.cfg.MessageWaitTimeout = timeout
}

func (s *TeamsTester) Timeout() time.Duration {
	return s.cfg.MessageWaitTimeout
}

func (s *TeamsTester) CreateChannel(t *testing.T, prefix string) (Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s-%s", channelNamePrefix, prefix, randomID.String())

	t.Logf("Creating channel %q...", channelName)
	ctx := context.Background()
	channelID, err := s.cli.CreateChannel(ctx, s.cfg.TeamID, channelName)
	require.NoError(t, err)

	t.Logf("Channel %q (ID: %q) created", channelName, channelID)

	cleanupFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Archiving channel %q...", channelName)
		err = s.cli.DeleteChannel(ctx, s.cfg.TeamID, channelID)
		assert.NoError(t, err)
	}

	return &TeamsChannel{id: channelID, name: channelName}, cleanupFn
}

func (s *TeamsConfig) BotUsername() string {
	if s.CloudBasedTestEnabled {
		return s.CloudBotName
	}
	return s.BotName
}

func (s *TeamsTester) trimAttachmentTimestamp(in string) (string, string) {
	msgParts := strings.Split(in, " <!date^")
	ts := ""
	if len(msgParts) > 1 {
		ts = strings.Split(msgParts[1], "^")[0]
	}
	return msgParts[0], ts
}
