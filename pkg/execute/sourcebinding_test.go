package execute

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	groupName = "testing-source-bindings"
	platform  = config.SlackCommPlatformIntegration
	userID    = "Joe"
	botName   = "Botkube"
)

var (
	conversation = Conversation{ID: "id", Alias: "alias"}
)

func TestSourceBindingsHappyPath(t *testing.T) {
	cfg := config.Config{
		Sources: map[string]config.Sources{
			"bar": {
				DisplayName: "BAR",
			},
			"xyz": {
				DisplayName: "XYZ",
			},
			"fiz": {
				DisplayName: "FIZ",
			},
			"foo": {
				DisplayName: "FOO",
			},
			"baz": {
				DisplayName: "BAZ",
			},
		},
		ConfigWatcher: config.CfgWatcher{
			Enabled: true,
		},
	}
	cfgWithCfgWatcherDisabled := config.Config{Sources: cfg.Sources}

	tests := []struct {
		name    string
		command string
		config  config.Config

		message        string
		sourceBindings []string
	}{
		{
			name:    "Should resolve quoted list which is separated by comma",
			command: `edit SourceBindings "bar,xyz"`,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR` and `XYZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve quoted and code items separated by comma",
			command: "edit sourcebindings “`bar`,xyz ”",
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR` and `XYZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma and ends with whitespace",
			command: `edit sourceBindings bar,xyz `,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR` and `XYZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma but has a lot of whitespaces",
			command: `edit sourcebindings bar,       xyz, `,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR` and `XYZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma, has a lot of whitespaces and some items are quoted",
			command: `edit SourceBindings bar       xyz, "baz"`,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR`, `XYZ`, and `BAZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"bar", "xyz", "baz"},
		},
		{
			name:    "Should resolve list with unicode quotes",
			command: `edit SourceBindings “foo,bar”`,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `FOO` and `BAR` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"foo", "bar"},
		},
		{
			name:    "Should resolve list which has mixed formatting for different items, all at once",
			command: `edit SourceBindings foo baz "bar,xyz" "fiz"`,
			config:  cfg,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `FOO`, `BAZ`, `BAR`, `XYZ`, and `FIZ` messages for this channel. Expect Botkube reload in a few seconds...",
			sourceBindings: []string{"foo", "baz", "bar", "xyz", "fiz"},
		},
		{
			name:    "Should mention manual app restart",
			command: `edit SourceBindings "bar,xyz"`,
			config:  cfgWithCfgWatcherDisabled,

			message:        ":white_check_mark: Joe adjusted the Botkube notifications settings to `BAR` and `XYZ` messages.\nAs the Config Watcher is disabled, you need to restart Botkube manually to apply the changes.",
			sourceBindings: []string{"bar", "xyz"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			fakeStorage := &fakeBindingsStorage{}
			args := strings.Fields(strings.TrimSpace(tc.command))
			executor := NewSourceBindingExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, fakeStorage, tc.config)
			cmdCtx := CommandContext{
				Args:          args,
				CommGroupName: groupName,
				Platform:      platform,
				Conversation:  conversation,
				User:          userID,
				BotName:       botName,
			}
			expMessage := interactive.CoreMessage{
				Description: tc.message,
			}
			// when
			msg, err := executor.Edit(context.Background(), cmdCtx)

			// then
			require.NoError(t, err)
			assert.Equal(t, expMessage, msg)
			assert.Equal(t, tc.sourceBindings, fakeStorage.sourceBindings)
			assert.Equal(t, groupName, fakeStorage.commGroupName)
			assert.Equal(t, platform, fakeStorage.platform)
			assert.Equal(t, conversation.Alias, fakeStorage.channelAlias)
		})
	}
}

func TestSourceBindingsErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
		expErr  error
		expMsg  interactive.CoreMessage
	}{
		{
			name:    "Unknown source name",
			command: `edit SourceBindings something-else`,

			expErr: nil,
			expMsg: interactive.CoreMessage{
				Description: ":exclamation: The `something-else` source was not found in configuration. To learn how to add custom source, visit https://docs.botkube.io/configuration/source.",
			},
		},
		{
			name:    "Multiple unknown source names",
			command: `edit SourceBindings something-else other`,

			expErr: nil,
			expMsg: interactive.CoreMessage{
				Description: ":exclamation: The `something-else` and `other` sources were not found in configuration. To learn how to add custom source, visit https://docs.botkube.io/configuration/source.",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			args := strings.Fields(strings.TrimSpace(tc.command))
			executor := NewSourceBindingExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, nil, config.Config{})
			cmdCtx := CommandContext{
				Args:          args,
				CommGroupName: groupName,
				Platform:      platform,
				Conversation:  conversation,
				User:          userID,
				BotName:       botName,
			}
			// when
			msg, err := executor.Edit(context.Background(), cmdCtx)

			// then
			assert.ErrorIs(t, err, tc.expErr)
			assert.Equal(t, tc.expMsg, msg)
		})
	}
}

func TestSourceBindingsMultiSelectMessage(t *testing.T) {
	// given
	args := strings.Fields(strings.TrimSpace(`edit SourceBindings`))
	cfg := config.Config{
		Sources: map[string]config.Sources{
			"bar": {DisplayName: "BAR"},
			"xyz": {DisplayName: "XYZ"},
			"fiz": {DisplayName: "FIZ"},
			"foo": {DisplayName: "FOO"},
			"baz": {DisplayName: "BAZ"},
		},
		Communications: map[string]config.Communications{
			groupName: {
				Slack: config.Slack{
					Channels: config.IdentifiableMap[config.ChannelBindingsByName]{
						conversation.ID: config.ChannelBindingsByName{
							Name: conversation.ID,
							Bindings: config.BotBindings{
								Sources: []string{"bar", "fiz", "baz"},
							},
						},
					},
				},
			},
		},
	}

	expMsg := interactive.CoreMessage{
		Header: "Adjust notifications",
		Message: api.Message{

			Type:              api.PopupMessage,
			OnlyVisibleForYou: true,
			Sections: []api.Section{
				{
					MultiSelect: api.MultiSelect{
						Name: "Adjust notifications",
						Description: api.Body{
							Plaintext: "Select notification sources.",
						},
						Command: "Botkube edit SourceBindings",
						Options: []api.OptionItem{
							{Name: "BAR", Value: "bar"},
							{Name: "BAZ", Value: "baz"},
							{Name: "FIZ", Value: "fiz"},
							{Name: "FOO", Value: "foo"},
							{Name: "XYZ", Value: "xyz"},
						},
						InitialOptions: []api.OptionItem{
							{Name: "BAR", Value: "bar"},
							{Name: "FIZ", Value: "fiz"},
							{Name: "BAZ", Value: "baz"},
						},
					},
				},
			},
		},
	}

	executor := NewSourceBindingExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, nil, cfg)
	cmdCtx := CommandContext{
		Args:          args,
		CommGroupName: groupName,
		Platform:      platform,
		Conversation:  conversation,
		User:          userID,
		BotName:       botName,
	}
	// when
	gotMsg, err := executor.Edit(context.Background(), cmdCtx)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expMsg, gotMsg)
}

func TestSourceBindingsMultiSelectMessageWithIncorrectBindingConfig(t *testing.T) {
	// given
	args := strings.Fields(strings.TrimSpace(`edit SourceBindings`))
	cfg := config.Config{
		Sources: map[string]config.Sources{
			"bar": {DisplayName: "BAR"},
			"xyz": {DisplayName: "XYZ"},
		},
		Communications: map[string]config.Communications{
			groupName: {
				Slack: config.Slack{
					Channels: config.IdentifiableMap[config.ChannelBindingsByName]{
						conversation.ID: config.ChannelBindingsByName{
							Name: conversation.ID,
							Bindings: config.BotBindings{
								Sources: []string{"unknown", "source", "test"},
							},
						},
					},
				},
			},
		},
	}

	expMsg := interactive.CoreMessage{
		Header: "Adjust notifications",
		Message: api.Message{
			Type:              api.PopupMessage,
			OnlyVisibleForYou: true,
			Sections: []api.Section{
				{
					MultiSelect: api.MultiSelect{
						Name: "Adjust notifications",
						Description: api.Body{
							Plaintext: "Select notification sources.",
						},
						Command: "Botkube edit SourceBindings",
						Options: []api.OptionItem{
							{Name: "BAR", Value: "bar"},
							{Name: "XYZ", Value: "xyz"},
						},
					},
				},
			},
		},
	}

	executor := NewSourceBindingExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, nil, cfg)
	cmdCtx := CommandContext{
		Args:          args,
		CommGroupName: groupName,
		Platform:      platform,
		Conversation:  conversation,
		User:          userID,
		BotName:       botName,
	}
	// when
	gotMsg, err := executor.Edit(context.Background(), cmdCtx)

	// then
	assert.NoError(t, err)
	assert.EqualValues(t, expMsg, gotMsg)
}

type fakeBindingsStorage struct {
	commGroupName  string
	platform       config.CommPlatformIntegration
	channelAlias   string
	sourceBindings []string
}

func (f *fakeBindingsStorage) PersistSourceBindings(_ context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, sourceBindings []string) error {
	f.commGroupName = commGroupName
	f.platform = platform
	f.channelAlias = channelAlias
	f.sourceBindings = sourceBindings
	return nil
}
