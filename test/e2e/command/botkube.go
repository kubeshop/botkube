package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/test/e2e/utils"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type botkubeCommand struct {
	command  string
	expected string
}

// Send botkube command via Slack message and check if BotKube returns correct response
func (c *context) testBotkubeCommand(t *testing.T) {
	botkubeVersion := os.Getenv("BOTKUBE_VERSION")
	// Test cases
	tests := map[string]botkubeCommand{
		"BotKube ping": {
			command:  "ping",
			expected: fmt.Sprintf("```pong from cluster '%s'\n\nK8s Server Version: %s\nBotKube version: %s```", c.Config.Settings.ClusterName, execute.K8sVersion, botkubeVersion),
		},
		"BotKube filters list": {
			command: "filters list",
			expected: "FILTER                  ENABLED DESCRIPTION\n" +
				"NamespaceChecker        true    Checks if event belongs to blocklisted namespaces and filter them.\n" +
				"NodeEventsChecker       true    Sends notifications on node level critical events.\n" +
				"ObjectAnnotationChecker true    Checks if annotations botkube.io/* present in object specs and filters them.\n" +
				"PodLabelChecker         true    Checks and adds recommedations if labels are missing in the pod specs.\n" +
				"ImageTagChecker         true    Checks and adds recommendation if 'latest' image tag is used for container image.\n" +
				"IngressValidator        true    Checks if services and tls secrets used in ingress specs are available.\n" +
				"JobStatusChecker        true    Sends notifications only when job succeeds and ignores other job update events.\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Send message to a channel
			c.SlackServer.SendMessageToBot(c.Config.Communications.Slack.Channel, test.command)

			// Get last seen slack message
			time.Sleep(time.Second)
			lastSeenMsg := c.GetLastSeenSlackMessage()

			// Convert text message into Slack message structure
			m := slack.Message{}
			err := json.Unmarshal([]byte(lastSeenMsg), &m)
			assert.NoError(t, err, "message should decode properly")
			assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
			switch test.command {
			case "filters list":
				fl := compareFilters(strings.Split(test.expected, "\n"), strings.Split(strings.Trim(m.Text, "```"), "\n"))
				assert.Equal(t, fl, true)
			default:
				assert.Equal(t, test.expected, m.Text)
			}
		})
	}
}

func compareFilters(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// Compare slices
	for _, a := range actual {
		found := false
		for _, e := range expected {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Test disable notification with BotKube notifier command
// - disable notifier with '@BotKube notifier stop'
// - create pod and verify BotKube doesn't send notification
// - enable notifier with '@BotKube notifier start'
func (c *context) testNotifierCommand(t *testing.T) {
	// Disable notifier with @BotKube notifier stop
	t.Run("disable notifier", func(t *testing.T) {
		// Send message to a channel
		c.SlackServer.SendMessageToBot(c.Config.Communications.Slack.Channel, "notifier stop")

		// Get last seen slack message
		time.Sleep(time.Second)
		lastSeenMsg := c.GetLastSeenSlackMessage()

		// Convert text message into Slack message structure
		m := slack.Message{}
		err := json.Unmarshal([]byte(lastSeenMsg), &m)
		assert.NoError(t, err, "message should decode properly")
		assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
		assert.Equal(t, fmt.Sprintf("```Sure! I won't send you notifications from cluster '%s' anymore.```", c.Config.Settings.ClusterName), m.Text)
		assert.Equal(t, config.Notify, false)
	})

	// Create pod and verify that BotKube is not sending notifications
	pod := utils.CreateObjects{
		Kind:      "pod",
		Namespace: "test",
		Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-notifier"}},
		Expected: utils.SlackMessage{
			Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Pod create", Value: "Pod `test-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
		},
	}
	t.Run("create resource", func(t *testing.T) {
		// Inject an event into the fake client.
		utils.CreateResource(t, pod)

		// Get last seen slack message
		time.Sleep(time.Second)
		lastSeenMsg := c.GetLastSeenSlackMessage()

		// Convert text message into Slack message structure
		m := slack.Message{}
		err := json.Unmarshal([]byte(lastSeenMsg), &m)
		assert.NoError(t, err, "message should decode properly")
		assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
		assert.NotEqual(t, pod.Expected.Attachments, m.Attachments)
	})

	// Revert and Enable notifier
	t.Run("Enable notifier", func(t *testing.T) {
		// Send message to a channel
		c.SlackServer.SendMessageToBot(c.Config.Communications.Slack.Channel, "notifier start")

		// Get last seen slack message
		time.Sleep(time.Second)
		lastSeenMsg := c.GetLastSeenSlackMessage()

		// Convert text message into Slack message structure
		m := slack.Message{}
		err := json.Unmarshal([]byte(lastSeenMsg), &m)
		assert.NoError(t, err, "message should decode properly")
		assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
		assert.Equal(t, fmt.Sprintf("```Brace yourselves, notifications are coming from cluster '%s'.```", c.Config.Settings.ClusterName), m.Text)
		assert.Equal(t, config.Notify, true)
	})
}
