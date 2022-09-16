package e2e

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/botkube/pkg/multierror"
)

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
}

func newDiscordDriver(discordCfg DiscordConfig) (BotDriver, error) {
	discordCli, err := discordgo.New("Bot " + discordCfg.TesterAppToken)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}
	return &discordTester{cli: discordCli, cfg: discordCfg}, nil
}

func (d *discordTester) Type() DriverType {
	return DiscordBot
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
	d.testerUserID = d.findUserID(t, d.cfg.TesterName)
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
	return d.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) bool {
		return strings.EqualFold(msg, expectedMsg)
	})
}

func (d *discordTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg string) bool {
		return strings.Contains(msg, expectedMsgSubstring)
	})
}

func (d *discordTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	return d.WaitForMessagePosted(userID, channelID, 1, func(msg string) bool {
		return msg == expectedMsg
	})
}

func (d *discordTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
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

			if !assertFn(msg.Content) {
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

func (d *discordTester) WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	return d.WaitForMessagePosted(userID, channelID, limitMessages, assertFn)
}

func (d *discordTester) WaitForMessagePostedWithAttachment(userID, channelID string, assertFn AttachmentAssertion) error {
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

			if len(msg.Embeds) != 1 {
				lastErr = err
				return false, nil
			}

			embed := msg.Embeds[0]

			if !assertFn(embed.Title, strconv.Itoa(embed.Color), embed.Description) {
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

func (d *discordTester) WaitForMessagesPostedOnChannelsWithAttachment(userID string, channelIDs []string, assertFn AttachmentAssertion) error {
	errs := multierror.New()
	for _, channelID := range channelIDs {
		errs = multierror.Append(errs, d.WaitForMessagePostedWithAttachment(userID, channelID, assertFn))
	}

	return errs.ErrorOrNil()
}

func (d *discordTester) WaitForInteractiveMessagePostedRecentlyEqual(userID, channelID string, _ interactive.Message) error {
	return d.WaitForMessagePosted(userID, channelID, recentMessagesLimit, func(msg string) bool {
		return true
	})
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
