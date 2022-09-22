package execute

import (
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	groupName      = "testing-source-bindings"
	platform       = config.SlackCommPlatformIntegration
	conversationID = "random"
	userID         = "Joe"
	botName        = "BotKube"
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
	}

	tests := []struct {
		name    string
		command string

		message        string
		sourceBindings []string
	}{
		{
			name:    "Should resolve quoted list which is separated by comma",
			command: `edit SourceBindings "bar,xyz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to BAR and XYZ messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma and ends with whitespace",
			command: `edit sourceBindings bar,xyz `,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to BAR and XYZ messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma but has a lot of whitespaces",
			command: `edit sourcebindings bar,       xyz, `,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to BAR and XYZ messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma, has a lot of whitespaces and some items are quoted",
			command: `edit SourceBindings bar       xyz, "baz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to BAR, XYZ, and BAZ messages.",
			sourceBindings: []string{"bar", "xyz", "baz"},
		},
		{
			name:    "Should resolve list with unicode quotes",
			command: `edit SourceBindings “foo,bar”`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to FOO and BAR messages.",
			sourceBindings: []string{"foo", "bar"},
		},
		{
			name:    "Should resolve list which has mixed formatting for different items, all at once",
			command: `edit SourceBindings foo baz "bar,xyz" "fiz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to FOO, BAZ, BAR, XYZ, and FIZ messages.",
			sourceBindings: []string{"foo", "baz", "bar", "xyz", "fiz"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			log, _ := logtest.NewNullLogger()

			fakeStorage := &fakeBindingsStorage{}
			args := strings.Fields(strings.TrimSpace(tc.command))
			executor := NewEditExecutor(log, &fakeAnalyticsReporter{}, fakeStorage, cfg)

			expMessage := interactive.Message{
				Base: interactive.Base{
					Description: tc.message,
				},
			}
			// when
			msg, err := executor.Do(args, groupName, platform, conversationID, userID, botName)

			// then
			require.NoError(t, err)
			assert.Equal(t, expMessage, msg)
			assert.Equal(t, tc.sourceBindings, fakeStorage.sourceBindings)
			assert.Equal(t, groupName, fakeStorage.commGroupName)
			assert.Equal(t, platform, fakeStorage.platform)
			assert.Equal(t, conversationID, fakeStorage.channelName)
		})
	}
}

func TestSourceBindingsErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
		expErr  error
		expMsg  interactive.Message
	}{
		{
			name:    "Wrong resource name",
			command: `edit Source Bindings "bar,xyz"`,

			expErr: errUnsupportedCommand,
		},
		{
			name:    "Typo in resource name",
			command: `edit SourceBindnigs bar,xyz`,

			expErr: errUnsupportedCommand,
		},
		{
			name:    "Unknown source name",
			command: `edit SourceBindings something-else`,

			expErr: nil,
			expMsg: interactive.Message{
				Base: interactive.Base{
					Description: ":X: The something-else source was not found in configuration.",
				},
			},
		},
		{
			name:    "Multiple unknown source names",
			command: `edit SourceBindings something-else other`,

			expErr: nil,
			expMsg: interactive.Message{
				Base: interactive.Base{
					Description: ":X: The something-else and other sources were not found in configuration.",
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			log, _ := logtest.NewNullLogger()

			args := strings.Fields(strings.TrimSpace(tc.command))
			executor := NewEditExecutor(log, &fakeAnalyticsReporter{}, nil, config.Config{})

			// when
			msg, err := executor.Do(args, groupName, platform, conversationID, userID, botName)

			// then
			assert.ErrorIs(t, err, tc.expErr)
			assert.Equal(t, tc.expMsg, msg)
		})
	}
}

func TestSourceBindingsMultiSelectMessage(t *testing.T) {
	t.Skip()
	// given
	log, _ := logtest.NewNullLogger()

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
						conversationID: config.ChannelBindingsByName{
							Name: conversationID,
							Bindings: config.BotBindings{
								Sources: []string{"bar", "fiz", "baz"},
							},
						},
					},
				},
			},
		},
	}

	expMsg := interactive.Message{
		Type: interactive.Popup,
		Base: interactive.Base{
			Header: "Adjust notifications",
		},
		Sections: []interactive.Section{
			{
				MultiSelect: interactive.MultiSelect{
					Name: "Adjust notifications",
					Description: interactive.Body{
						Plaintext: "Select notification sources.",
					},
					Command: "BotKube edit SourceBindings",
					Options: []interactive.OptionItem{
						{Name: "BAZ", Value: "baz"},
						{Name: "BAR", Value: "bar"},
						{Name: "XYZ", Value: "xyz"},
						{Name: "FIZ", Value: "fiz"},
						{Name: "FOO", Value: "foo"},
					},
					InitialOptions: []interactive.OptionItem{
						{Name: "BAR", Value: "bar"},
						{Name: "FIZ", Value: "fiz"},
						{Name: "BAZ", Value: "baz"},
					},
				},
			},
		},
	}

	executor := NewEditExecutor(log, &fakeAnalyticsReporter{}, nil, cfg)

	// when
	gotMsg, err := executor.Do(args, groupName, platform, conversationID, userID, botName)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expMsg, gotMsg)
}

type fakeBindingsStorage struct {
	commGroupName  string
	platform       config.CommPlatformIntegration
	channelName    string
	sourceBindings []string
}

func (f *fakeBindingsStorage) PersistSourceBindings(commGroupName string, platform config.CommPlatformIntegration, channelName string, sourceBindings []string) error {
	f.commGroupName = commGroupName
	f.platform = platform
	f.channelName = channelName
	f.sourceBindings = sourceBindings
	return nil
}
