package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscord_FindAndTrimBotMention(t *testing.T) {
	/// given
	botID := "976786722706821120"
	testCases := []struct {
		Name               string
		Input              string
		ExpectedTrimmedMsg string
		ExpectedFound      bool
	}{
		{
			Name:               "Mention",
			Input:              "<@976786722706821120> get pods",
			ExpectedFound:      true,
			ExpectedTrimmedMsg: " get pods",
		},
		{
			Name:               "Nickname",
			Input:              "<@!976786722706821120> get pods",
			ExpectedFound:      true,
			ExpectedTrimmedMsg: " get pods",
		},
		{
			Name:          "Not at the beginning",
			Input:         "Not at the beginning <@!976786722706821120> get pods",
			ExpectedFound: false,
		},
		{
			Name:          "Different mention",
			Input:         "<@97678> get pods",
			ExpectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			botMentionRegex, err := discordBotMentionRegex(botID)
			require.NoError(t, err)
			b := &Discord{botMentionRegex: botMentionRegex}

			// when
			actualTrimmedMsg, actualFound := b.findAndTrimBotMention(tc.Input)

			// then
			assert.Equal(t, tc.ExpectedFound, actualFound)
			assert.Equal(t, tc.ExpectedTrimmedMsg, actualTrimmedMsg)
		})
	}
}
