//go:build integration

package e2e

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/utils"
)

const recentMessagesLimit = 5

type slackTester struct {
	cli *slack.Client
	cfg SlackConfig
}

func newSlackTester(slackCfg SlackConfig) (*slackTester, error) {
	slackCli := slack.New(slackCfg.TesterAppToken)
	_, err := slackCli.AuthTest()
	if err != nil {
		return nil, err
	}

	return &slackTester{cli: slackCli, cfg: slackCfg}, nil
}

func (s *slackTester) CreateChannel(t *testing.T) (*slack.Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s", channelNamePrefix, randomID.String())

	t.Logf("Creating channel %q...", channelName)
	// > There’s no limit to how many unique channels you can have in Slack — go ahead, create as many as you’d like!
	// Sure, thanks Slack!
	// Source: https://slack.com/help/articles/201402297-Create-a-channel
	channel, err := s.cli.CreateConversation(channelName, false)
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

func (s *slackTester) PostMessageToBot(t *testing.T, channelName, command string) {
	message := fmt.Sprintf("<@%s> %s", s.cfg.BotName, command)
	_, _, err := s.cli.PostMessage(channelName, slack.MsgOptionText(message, false))
	require.NoError(t, err)
}

func (s *slackTester) FindUserIDForBot(t *testing.T) string {
	return s.FindUserID(t, s.cfg.BotName)
}

func (s *slackTester) FindUserIDForTester(t *testing.T) string {
	return s.FindUserID(t, s.cfg.TesterName)
}

func (s *slackTester) FindUserID(t *testing.T, name string) string {
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

func (s *slackTester) InviteBotToChannel(t *testing.T, botID, channelID string) {
	t.Logf("Inviting bot with ID %q to the channel with ID %q", botID, channelID)
	_, err := s.cli.InviteUsersToConversation(channelID, botID)
	require.NoError(t, err)
}

func (s *slackTester) WaitForMessagePostedRecentlyEqual(userID, channelID string, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg slack.Message) bool {
		return strings.EqualFold(msg.Text, expectedMsg)
	})
}

func (s *slackTester) WaitForLastMessageContains(userID, channelID string, expectedMsgSubstring string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg slack.Message) bool {
		return strings.Contains(msg.Text, expectedMsgSubstring)
	})
}

func (s *slackTester) WaitForLastMessageEqual(userID, channelID string, expectedMsg string) error {
	return s.WaitForMessagePosted(userID, channelID, 1, func(msg slack.Message) bool {
		return msg.Text == expectedMsg
	})
}

func (s *slackTester) WaitForLastMessageEqualOnChannels(userID string, channelIDs []string, expectedMsg string) error {
	return s.WaitForMessagesPostedOnChannels(userID, channelIDs, 1, func(msg slack.Message) bool {
		return msg.Text == expectedMsg
	})
}

func (s *slackTester) WaitForMessagePosted(userID, channelID string, limitMessages int, msgAssertFn func(msg slack.Message) bool) error {
	var fetchedMessages []slack.Message
	var lastErr error
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

			if !msgAssertFn(msg) {
				// different message
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
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, utils.StructDumper().Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *slackTester) WaitForMessagesPostedOnChannels(userID string, channelIDs []string, limitMessages int, msgAssertFn func(msg slack.Message) bool) error {
	errs := multierror.New()
	for _, channelID := range channelIDs {
		errs = multierror.Append(errs, s.WaitForMessagePosted(userID, channelID, limitMessages, msgAssertFn))
	}

	return errs.ErrorOrNil()
}
