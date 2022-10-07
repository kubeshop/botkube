package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlack_FindAndTrimBotMention(t *testing.T) {
	/// given
	botName := "Botkube"
	testCases := []struct {
		Name               string
		Input              string
		ExpectedTrimmedMsg string
		ExpectedFound      bool
	}{
		{
			Name:               "Mention",
			Input:              "<@Botkube> get pods",
			ExpectedFound:      true,
			ExpectedTrimmedMsg: " get pods",
		},
		{
			Name:          "Not at the beginning",
			Input:         "Not at the beginning <@Botkube> get pods",
			ExpectedFound: false,
		},
		{
			Name:          "Different mention",
			Input:         "<@bootkube> get pods",
			ExpectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			botMentionRegex, err := slackBotMentionRegex(botName)
			require.NoError(t, err)
			b := &Slack{botMentionRegex: botMentionRegex}
			require.NoError(t, err)

			// when
			actualTrimmedMsg, actualFound := b.findAndTrimBotMention(tc.Input)

			// then
			assert.Equal(t, tc.ExpectedFound, actualFound)
			assert.Equal(t, tc.ExpectedTrimmedMsg, actualTrimmedMsg)
		})
	}
}
