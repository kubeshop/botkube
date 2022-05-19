package bot_test

import (
	"fmt"
	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSlackBot_StripUnmarshallingErrEventDetails(t *testing.T) {
	//given
	sampleEvent := `{"type":"user_huddle_changed","user":{"id":"id","team_id":"team_id"}, "event_ts":"1652949120.004700"}`

	testCases := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "Unmapped event",
			Input:    fmt.Sprintf(`RTM Error: Received unmapped event "user_huddle_changed": %s`, sampleEvent),
			Expected: `RTM Error: Received unmapped event "user_huddle_changed"`,
		},
		{
			Name:     "Unmarshalling error message",
			Input:    fmt.Sprintf(`RTM Error: Could not unmarshall event "user_huddle_changed": %s`, sampleEvent),
			Expected: `RTM Error: Could not unmarshall event "user_huddle_changed"`,
		},
		{
			Name:     "JSON unmarshal error",
			Input:    "cannot unmarshal bool into Go value of type string",
			Expected: "cannot unmarshal bool into Go value of type string",
		},
		{
			Name: "JSON unmarshal error with colons",
			// this is a real error when doing json.Unmarshal([]byte(`":::"`), &time)
			Input:    `parsing time "":::"" as ""2006-01-02T15:04:05Z07:00"": cannot parse ":::"" as "2006"`,
			Expected: `parsing time "":::"" as ""2006-01-02T15:04:05Z07:00"": cannot parse ":::"" as "2006"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			slackBot := &bot.SlackBot{}

			// when
			actual := slackBot.StripUnmarshallingErrEventDetails(testCase.Input)

			// then
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}
