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

func TestSourceBindingsHappyPath(t *testing.T) {
	const (
		groupName      = "testing-source-bindings"
		platform       = config.SlackCommPlatformIntegration
		conversationID = "random"
		userID         = "Joe"
	)
	tests := []struct {
		name    string
		command string

		message        string
		sourceBindings []string
	}{
		{
			name:    "Should resolve quoted list which is separated by comma",
			command: `edit SourceBindings "bar,xyz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to bar and xyz messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma and ends with whitespace",
			command: `edit sourceBindings bar,xyz `,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to bar and xyz messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma but has a lot of whitespaces",
			command: `edit sourcebindings bar,       xyz, `,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to bar and xyz messages.",
			sourceBindings: []string{"bar", "xyz"},
		},
		{
			name:    "Should resolve list which is separated by comma, has a lot of whitespaces and some items are quoted",
			command: `edit SourceBindings bar       xyz, "baz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to bar, xyz, and baz messages.",
			sourceBindings: []string{"bar", "xyz", "baz"},
		},
		{
			name:    "Should resolve list with unicode quotes",
			command: `edit SourceBindings “foo,bar”`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to foo and bar messages.",
			sourceBindings: []string{"foo", "bar"},
		},
		{
			name:    "Should resolve list which has mixed formatting for different items, all at once",
			command: `edit SourceBindings foo baz "bar,xyz" "fiz"`,

			message:        ":white_check_mark: Joe adjusted the BotKube notifications settings to foo, baz, bar, xyz, and fiz messages.",
			sourceBindings: []string{"foo", "baz", "bar", "xyz", "fiz"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			log, _ := logtest.NewNullLogger()

			fakeStorage := &fakeBindingsStorage{}
			args := strings.Fields(strings.TrimSpace(tc.command))
			executor := NewEditExecutor(log, &fakeAnalyticsReporter{}, fakeStorage)

			expMessage := interactive.Message{
				Base: interactive.Base{
					Description: tc.message,
				},
			}
			// when
			msg, err := executor.Do(args, groupName, platform, conversationID, userID)

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
