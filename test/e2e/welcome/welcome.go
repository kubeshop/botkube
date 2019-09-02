package welcome

import (
	"encoding/json"
	"testing"

	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends welcome message to the configured slack channel after start
func (c *context) testWelcome(t *testing.T) {
	expected := "...and now my watch begins for cluster 'test-cluster-1'! :crossed_swords:"

	if c.TestEnv.Config.Communications.Slack.Enabled {

		// Get last seen slack message
		lastSeenMsg := c.GetLastSeenSlackMessage()

		// Convert text message into Slack message structure
		m := slack.Message{}
		err := json.Unmarshal([]byte(*lastSeenMsg), &m)
		assert.NoError(t, err, "message should decode properly")
		assert.Equal(t, c.TestEnv.Config.Communications.Slack.Channel, m.Channel)
		assert.Equal(t, expected, m.Text)
	}
}

// E2ETests run welcome tests
func E2ETests(testEnv *env.TestEnv) func(*testing.T) {
	ctx := &context{
		testEnv,
	}

	return ctx.testWelcome
}
