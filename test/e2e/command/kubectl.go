package command

import (
	"encoding/json"
	"fmt"
	"testing"

	exec "github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type kubectlCommand struct {
	command  string
	expected string
}

type context struct {
	*env.TestEnv
}

// Send kubectl command via Slack message and check if BotKube returns correct response
func (c *context) testKubectlCommand(t *testing.T) {
	// Test cases
	tests := map[string]kubectlCommand{
		"BotKube get pods": {
			command:  "get pods",
			expected: fmt.Sprintf("```Cluster: %s\n%s```", c.Config.Settings.ClusterName, exec.KubectlResponse["-n default get pods"]),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if c.TestEnv.Config.Communications.Slack.Enabled {
				// Send message to a channel
				c.SlackServer.SendMessageToBot(c.Config.Communications.Slack.Channel, test.command)

				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage()

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
				assert.Equal(t, test.expected, m.Text)
			}
		})
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	// Run kubectl tests
	t.Run("Test Kubectl command", c.testKubectlCommand)
	t.Run("Test BotKube command", c.testBotkubeCommand)
	t.Run("Test disable notifier", c.testNotifierCommand)
}

// E2ETests runs command execution tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
