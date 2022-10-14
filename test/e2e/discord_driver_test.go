//go:build integration

package e2e

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

var DiscordAttachmentColorStatus = AttachmentStatus{
	config.Info:     "8311585",
	config.Debug:    "8311585",
	config.Warn:     "16312092",
	config.Error:    "13632027",
	config.Critical: "13632027",
}

type DiscordChannel struct {
	*discordgo.Channel
}

func (s *DiscordChannel) ID() string {
	return s.Channel.ID
}
func (s *DiscordChannel) Name() string {
	return s.Channel.Name
}
func (s *DiscordChannel) Identifier() string {
	return s.Channel.ID
}

type discordTester struct {
	cli           *discordgo.Session
	cfg           DiscordConfig
	botUserID     string
	testerUserID  string
	channel       Channel
	secondChannel Channel
	mdFormatter   interactive.MDFormatter
}

func newDiscordDriver(discordCfg DiscordConfig) (BotDriver, error) {
	discordCli, err := discordgo.New("Bot " + discordCfg.TesterAppToken)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}
	return &discordTester{cli: discordCli, cfg: discordCfg, mdFormatter: interactive.DefaultMDFormatter()}, nil
}

func (d *discordTester) Type() DriverType {
	return DiscordBot
}

func (d *discordTester) BotName() string {
	return "@Botkube"
}

func (d *discordTester) BotUserID() string {
	return d.botUserID
}

func (d *discordTester) TesterUserID() string {
	return d.testerUserID
}

func (d *discordTester) Channel() Channel {
	return d.channel
}

func (d *discordTester) SecondChannel() Channel {
	return d.secondChannel
}

func (d *discordTester) InitUsers(t *testing.T) {
	t.Helper()
	d.botUserID = d.findUserID(t, d.cfg.BotName)
	assert.NotEmpty(t, d.botUserID, "could not find discord botUserID with name: %s", d.cfg.BotName)

	d.testerUserID = d.findUserID(t, d.cfg.TesterName)
	assert.NotEmpty(t, d.testerUserID, "could not find discord testerUserID with name: %s", d.cfg.TesterName)
}

func (d *discordTester) InitChannels(t *testing.T) []func() {
	channel, cleanupChannelFn := d.createChannel(t)
	d.channel = &DiscordChannel{Channel: channel}

	secondChannel, cleanupSecondChannelFn := d.createChannel(t)
	d.secondChannel = &DiscordChannel{Channel: secondChannel}

	return []func(){
		func() { cleanupChannelFn(t) },
		func() { cleanupSecondChannelFn(t) },
	}
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

func (d *discordTester) PostMessageToBot(t *testing.T, channel, command string) {
	message := fmt.Sprintf("<@%s> %s", d.botUserID, command)
	_, err := d.cli.ChannelMessageSend(channel, message)
	require.NoError(t, err)
}

func (d *discordTester) InviteBotToChannel(_ *testing.T, _ string) {
	// This is not required in Discord.
	// Bots can't "join" text channels because when you join a server you're already in every text channel.
	// See: https://stackoverflow.com/questions/60990748/making-discord-bot-join-leave-a-channel
}

func (d *discordTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	return d.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) (bool, int, string) {
		if !strings.EqualFold(expectedMsg, msg) {
			count := countMatchBlock(expectedMsg, msg)
			msgDiff := diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (d *discordTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		return strings.Contains(msg, expectedMsgSubstring), 0, ""
	})
}

func (d *discordTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		if msg != expectedMsg {
			count := countMatchBlock(expectedMsg, msg)
			msgDiff := diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (d *discordTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	// To always receive message content:
	// ensure you enable the MESSAGE CONTENT INTENT for the tester bot on the developer portal.
	// Applications ↦ Settings ↦ Bot ↦ Privileged Gateway Intents
	// This setting has been enforced from August 31, 2022

	var fetchedMessages []*discordgo.Message
	var lastErr error
	var highestCommonBlockCount int
	if limitMessages == 1 {
		highestCommonBlockCount = -1 // a single message is fetched, always print diff
	}
	var diffMessage string

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

			equal, commonCount, diffStr := assertFn(msg.Content)
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

func (d *discordTester) WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	return d.WaitForMessagePosted(userID, channelID, limitMessages, assertFn)
}

func (d *discordTester) WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error {
	// To always receive message content:
	// ensure you enable the MESSAGE CONTENT INTENT for the tester bot on the developer portal.
	// Applications ↦ Settings ↦ Bot ↦ Privileged Gateway Intents
	// This setting has been enforced from August 31, 2022

	var fetchedMessages []*discordgo.Message
	var lastErr error

	err := wait.Poll(pollInterval, d.cfg.MessageWaitTimeout, func() (done bool, err error) {
		messages, err := d.cli.ChannelMessages(channelID, 1, "", "", "")
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = messages
		for _, msg := range messages {
			if msg.Author.ID != userID {
				continue
			}

			if len(msg.Attachments) != 1 {
				lastErr = err
				return false, nil
			}

			upload := msg.Attachments[0]
			if !assertFn(upload.Filename, upload.ContentType) {
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

func (d *discordTester) WaitForMessagePostedWithAttachment(userID, channelID string, assertFn AttachmentAssertion) error {
	// To always receive message content:
	// ensure you enable the MESSAGE CONTENT INTENT for the tester bot on the developer portal.
	// Applications ↦ Settings ↦ Bot ↦ Privileged Gateway Intents
	// This setting has been enforced from August 31, 2022

	var fetchedMessages []*discordgo.Message
	var lastErr error
	var diffMessage string
	highestCommonBlockCount := -1 // a single message is fetched, always print diff

	err := wait.Poll(pollInterval, d.cfg.MessageWaitTimeout, func() (done bool, err error) {
		messages, err := d.cli.ChannelMessages(channelID, 1, "", "", "")
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = messages
		for _, msg := range messages {
			if msg.Author.ID != userID {
				continue
			}

			if len(msg.Embeds) != 1 {
				lastErr = err
				return false, nil
			}

			embed := msg.Embeds[0]

			equal, commonCount, diffStr := assertFn(embed.Title, strconv.Itoa(embed.Color), embed.Description)
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

func (d *discordTester) WaitForMessagesPostedOnChannelsWithAttachment(userID string, channelIDs []string, assertFn AttachmentAssertion) error {
	errs := multierror.New()
	for _, channelID := range channelIDs {
		errs = multierror.Append(errs, d.WaitForMessagePostedWithAttachment(userID, channelID, assertFn))
	}

	return errs.ErrorOrNil()
}

func (d *discordTester) WaitForInteractiveMessagePostedRecentlyEqual(userID, channelID string, msg interactive.Message) error {
	markdown := strings.TrimSpace(interactive.RenderMessage(d.mdFormatter, msg))
	return d.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) (bool, int, string) {
		if !strings.EqualFold(markdown, msg) {
			count := countMatchBlock(markdown, msg)
			msgDiff := diff(markdown, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (d *discordTester) WaitForLastInteractiveMessagePostedEqual(userID, channelID string, msg interactive.Message) error {
	markdown := strings.TrimSpace(interactive.RenderMessage(d.mdFormatter, msg))
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		if !strings.EqualFold(markdown, msg) {
			count := countMatchBlock(markdown, msg)
			msgDiff := diff(markdown, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (d *discordTester) GetColorByLevel(level config.Level) string {
	return DiscordAttachmentColorStatus[level]
}

func (d *discordTester) findUserID(t *testing.T, name string) string {
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

func (d *discordTester) createChannel(t *testing.T) (*discordgo.Channel, func(t *testing.T)) {
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
