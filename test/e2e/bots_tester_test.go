//go:build integration

package e2e

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
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

type discordTester struct {
	cli *discordgo.Session
	cfg DiscordConfig
}

func newSlackTester(slackCfg SlackConfig) (*slackTester, error) {
	slackCli := slack.New(slackCfg.TesterAppToken)
	_, err := slackCli.AuthTest()
	if err != nil {
		return nil, err
	}

	return &slackTester{cli: slackCli, cfg: slackCfg}, nil
}

func newDiscordTester(discordCfg DiscordConfig) (*discordTester, error) {
	discordCli, err := discordgo.New("Bot " + discordCfg.TesterAppToken)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}
	return &discordTester{cli: discordCli, cfg: discordCfg}, nil
}

func (d *discordTester) CreateChannel(t *testing.T) (*discordgo.Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s", channelNamePrefix, randomID.String())

	t.Logf("Creating channel %q...", channelName)
	channel, err := d.cli.GuildChannelCreate(d.cfg.GuildID, channelName, discordgo.ChannelTypeGuildText)
	require.NoError(t, err)

	t.Logf("Channel %q (ID: %q) created", channelName, channel.ID)

	cleanupFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Deleting channel %q...", channel.Name)
		// We cannot archive a channel: https://support.discord.com/hc/en-us/community/posts/360042842012-Archive-old-chat-channels
		_, err := d.cli.ChannelDelete(channel.ID)
		assert.NoError(t, err)
	}

	return channel, cleanupFn
}

func (d *discordTester) FindUserIDForBot(t *testing.T) string {
	return d.FindUserID(t, d.cfg.BotName)
}

func (d *discordTester) FindUserIDForTester(t *testing.T) string {
	return d.FindUserID(t, d.cfg.TesterName)
}

func (d *discordTester) FindUserID(t *testing.T, name string) string {
	t.Log("Getting users...")
	res, err := d.cli.GuildMembersSearch(d.cfg.GuildID, name, 5)
	require.NoError(t, err)

	t.Logf("Finding user ID by name %q...", name)
	for _, m := range res {
		if !strings.EqualFold(name, m.User.Username) {
			continue
		}
		return m.User.ID
	}

	return ""
}

func (d *discordTester) PostInitialMessage(t *testing.T, channelID string) {
	t.Helper()
	t.Logf("Posting welcome message for channel: %s...", channelID)

	var additionalContextMsg string
	if d.cfg.AdditionalContextMessage != "" {
		additionalContextMsg = fmt.Sprintf("%s\n", d.cfg.AdditionalContextMessage)
	}
	message := fmt.Sprintf("Hello!\n%s%s", additionalContextMsg, welcomeText)
	_, err := d.cli.ChannelMessageSend(channelID, message)
	require.NoError(t, err)
}

func (d *discordTester) InviteBotToChannel(_ *testing.T, _, _ string) {
	// This is not required in Discord.
	// Bots can't "join" text channels because when you join a server you're already in every text channel.
	// See: https://stackoverflow.com/questions/60990748/making-discord-bot-join-leave-a-channel
}

func (d *discordTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	return d.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg *discordgo.Message) bool {
		return strings.EqualFold(msg.Content, expectedMsg)
	})
}

func (d *discordTester) WaitForLastMessageContains(userID, channelID string, expectedMsgSubstring string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg *discordgo.Message) bool {
		return strings.Contains(msg.Content, expectedMsgSubstring)
	})
}

func (d *discordTester) WaitForLastMessageEqual(userID, channelID string, expectedMsg string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg *discordgo.Message) bool {
		return msg.Content == expectedMsg
	})
}

func (d *discordTester) PostMessageToBot(t *testing.T, userID, channelID, command string) {
	message := fmt.Sprintf("<@%s> %s", userID, command)
	_, err := d.cli.ChannelMessageSend(channelID, message)
	require.NoError(t, err)
}

func (d *discordTester) WaitForMessagePosted(userID, channelID string, limitMessages int, msgAssertFn func(msg *discordgo.Message) bool) error {
	// To always receive message content:
	// ensure you enable the MESSAGE CONTENT INTENT for the tester bot on the developer portal.
	// Applications ↦ Settings ↦ Bot ↦ Privileged Gateway Intents
	// This setting has been enforced from August 31, 2022

	var fetchedMessages []*discordgo.Message
	var lastErr error

	err := wait.Poll(pollInterval, d.cfg.MessageWaitTimeout, func() (done bool, err error) {
		messages, err := d.cli.ChannelMessages(channelID, limitMessages, "", "", "")
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = messages
		for _, msg := range messages {
			if msg.Author.ID != userID {
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
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (d *discordTester) WaitForMessagesPostedOnChannels(userID string, channelIDs []string, limitMessages int, msgAssertFn func(msg *discordgo.Message) bool) error {
	errs := multierror.New()
	for _, channelID := range channelIDs {
		errs = multierror.Append(errs, d.WaitForMessagePosted(userID, channelID, limitMessages, msgAssertFn))
	}

	return errs.ErrorOrNil()
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
