package welcome

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type context struct {
	Env *env.TestEnv
}

// Test if BotKube sends welcome message to the configured slack channel after start
func (c *context) testWelcome(t *testing.T) {
	expected := "...and now my watch begins for cluster 'test-cluster-1'! :crossed_swords:"

	// Get last seen slack message
	time.Sleep(time.Second)
	lastSeenMsg := c.Env.GetLastSeenSlackMessage()

	// Convert text message into Slack message structure
	m := slack.Message{}
	err := json.Unmarshal([]byte(*lastSeenMsg), &m)
	assert.NoError(t, err, "message should decode properly")
	assert.Equal(t, c.Env.Config.Communications.Slack.Channel, m.Channel)
	assert.Equal(t, expected, m.Text)
}

// E2ETests run welcome tests
func E2ETests(testEnv *env.TestEnv) func(*testing.T) {
	ctx := &context{
		Env: testEnv,
	}

	return ctx.testWelcome
}
