package bot

import (
	"testing"

	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/execute/command"
)

func TestExtractExplicitOrigin(t *testing.T) {
	tests := []struct {
		name       string
		givenAct   schema.Activity
		wantOrigin command.Origin
	}{
		{
			name: "Should return explicit origin",
			givenAct: schema.Activity{
				Type:  schema.Message,
				Value: explicitOriginValue(string(command.MultiSelectValueChangeOrigin)),
			},
			wantOrigin: command.MultiSelectValueChangeOrigin,
		},
		{
			name: "Should return typed message resolved from type due to invalid value",
			givenAct: schema.Activity{
				Type:  schema.Message,
				Value: explicitOriginValue("malformed-or-unknown-origin"),
			},
			wantOrigin: command.TypedOrigin,
		},
		{
			name: "Should return btn click origin resolved from type because value is nil",
			givenAct: schema.Activity{
				Type:  schema.Invoke,
				Value: nil,
			},
			wantOrigin: command.ButtonClickOrigin,
		},
		{
			name:       "Should return unknown origin because value does not contain origin key and type is empty",
			givenAct:   schema.Activity{Value: map[string]any{}},
			wantOrigin: command.UnknownOrigin,
		},
	}

	cloudTeam := &CloudTeams{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOrigin := cloudTeam.mapToCommandOrigin(tc.givenAct)
			assert.Equal(t, tc.wantOrigin, gotOrigin)
		})
	}
}

func TestTeams_TrimBotMention(t *testing.T) {
	/// given
	botName := "Botkube"
	testCases := []struct {
		Name               string
		Input              string
		ExpectedTrimmedMsg string
	}{
		{
			Name:               "Mention",
			Input:              "<at>Botkube</at> get pods",
			ExpectedTrimmedMsg: " get pods",
		},
		{
			Name:               "Not at the beginning",
			Input:              "Not at the beginning <at>Botkube</at> get pods",
			ExpectedTrimmedMsg: "Not at the beginning <at>Botkube</at> get pods",
		},
		{
			Name:               "Different mention",
			Input:              "<at>bootkube</at> get pods",
			ExpectedTrimmedMsg: "<at>bootkube</at> get pods",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			botMentionRegex, err := teamsBotMentionRegex(botName)
			require.NoError(t, err)
			b := &CloudTeams{botMentionRegex: botMentionRegex}
			require.NoError(t, err)

			// when
			actualTrimmedMsg := b.trimBotMention(tc.Input)

			// then
			assert.Equal(t, tc.ExpectedTrimmedMsg, actualTrimmedMsg)
		})
	}
}

func explicitOriginValue(in string) map[string]any {
	return map[string]any{
		originKeyName: in,
	}
}
