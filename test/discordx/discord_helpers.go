package discordx

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/bwmarrin/discordgo"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/stretchr/testify/require"
)

const (
	channelNamePrefix   = "test"
	pollInterval        = time.Second
	recentMessagesLimit = 2
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

type DiscordConfig struct {
	BotName                  string `envconfig:"optional"`
	BotID                    string `envconfig:"default=983294404108378154"`
	TesterName               string `envconfig:"optional"`
	TesterID                 string `envconfig:"default=1020384322114572381"`
	AdditionalContextMessage string `envconfig:"optional"`
	GuildID                  string
	TesterAppToken           string
	BotToken                 string
	MessageWaitTimeout       time.Duration `envconfig:"default=1m"`
}

type DiscordTester struct {
	cli           *discordgo.Session
	cfg           DiscordConfig
	botUserID     string
	testerUserID  string
	channel       Channel
	secondChannel Channel
	thirdChannel  Channel
	mdFormatter   interactive.MDFormatter
}

type Channel interface {
	ID() string
	Name() string
	Identifier() string
}

type MessageAssertion func(content string) bool

func New(discordCfg DiscordConfig) (*DiscordTester, error) {
	discordCli, err := discordgo.New("Bot " + discordCfg.TesterAppToken)
	if err != nil {
		return nil, fmt.Errorf("while creating Discord session: %w", err)
	}
	return &DiscordTester{cli: discordCli, cfg: discordCfg, mdFormatter: interactive.DefaultMDFormatter()}, nil
}
func (d *DiscordTester) InitUsers(t *testing.T) {
	t.Helper()

	d.botUserID = d.cfg.BotID
	if d.cfg.BotName != "" || d.botUserID == "" {
		t.Log("Bot user ID not set, looking for ID based on Bot name...")
		d.botUserID = d.findUserID(t, d.cfg.BotName)
		require.NotEmpty(t, d.botUserID, "could not find discord botUserID with name: %s", d.cfg.BotName)
	}

	d.testerUserID = d.cfg.TesterID
	if d.cfg.TesterName != "" || d.testerUserID == "" {
		t.Log("Tester user ID not set, looking for ID based on Tester name...")
		d.testerUserID = d.findUserID(t, d.cfg.TesterName)
		require.NotEmpty(t, d.testerUserID, "could not find discord testerUserID with name: %s", d.cfg.TesterName)
	}
}

func (d *DiscordTester) CreateChannel(t *testing.T, prefix string) (*discordgo.Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s-%s", channelNamePrefix, prefix, randomID.String())

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

func (d *DiscordTester) PostMessageToBot(t *testing.T, channel, command string) {
	message := fmt.Sprintf("<@%s> %s", d.botUserID, command)
	_, err := d.cli.ChannelMessageSend(channel, message)
	require.NoError(t, err)
}

func (d *DiscordTester) WaitForMessagePosted(userID, channelID string, assertFn MessageAssertion) error {
	var fetchedMessages []*discordgo.Message
	var lastErr error

	err := wait.Poll(pollInterval, d.cfg.MessageWaitTimeout, func() (done bool, err error) {
		messages, err := d.cli.ChannelMessages(channelID, recentMessagesLimit, "", "", "")
		if err != nil {
			lastErr = err
			return false, nil
		}

		fetchedMessages = messages
		for _, msg := range messages {
			if msg.Author.ID != userID {
				continue
			}

			expectedResult := assertFn(msg.Content)
			if !expectedResult {
				continue
			}

			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = fmt.Errorf("message assertion function returned false with %s", lastErr)
	}
	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, formatx.StructDumper().Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (d *DiscordTester) findUserID(t *testing.T, name string) string {
	t.Logf("Getting user %q...", name)
	res, err := d.cli.GuildMembersSearch(d.cfg.GuildID, name, 50)
	require.NoError(t, err)

	t.Logf("Finding user ID in %v...", res)
	for _, m := range res {
		if !strings.EqualFold(name, m.User.Username) {
			continue
		}
		return m.User.ID
	}

	return ""
}
