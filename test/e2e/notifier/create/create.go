package create

import (
	"encoding/json"
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends notification when a resource is created
func (c *context) testCreateResource(t *testing.T) {
	// Test cases
	tests := map[string]testutils.CreateObjects{
		"create pod in configured namespace": {
			Kind:      "pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Pod create", Value: "Pod `test-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: "", Messages: []string{"Resource created\n"}},
				Summary:     "Pod `test-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```",
			},
		},
		"create service in configured namespace": {
			Kind:      "service",
			Namespace: "test",
			Specs:     &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Service create", Value: "Service `test-service` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Service", Name: "test-service", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: "", Messages: []string{"Resource created\n"}},
				Summary:     "Service `test-service` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\n```",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Inject an event into the fake client.
			testutils.CreateResource(t, test)

			if c.TestEnv.Config.Communications.Slack.Enabled {

				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage()

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
				assert.Equal(t, test.ExpectedSlackMessage.Attachments, m.Attachments)
			}

			if c.TestEnv.Config.Communications.Webhook.Enabled {
				// Get last seen webhook payload
				lastSeenPayload := c.GetLastReceivedPayload()
				assert.Equal(t, test.ExpectedWebhookPayload.EventMeta, lastSeenPayload.EventMeta)
				assert.Equal(t, test.ExpectedWebhookPayload.EventStatus, lastSeenPayload.EventStatus)
				assert.Equal(t, test.ExpectedWebhookPayload.Summary, lastSeenPayload.Summary)
			}

			isAllowed := utils.AllowedEventKindsMap[utils.EventKind{Resource: test.Kind, Namespace: "all", EventType: config.CreateEvent}] ||
				utils.AllowedEventKindsMap[utils.EventKind{Resource: test.Kind, Namespace: test.Namespace, EventType: config.CreateEvent}]
			assert.Equal(t, isAllowed, true)
		})
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("create resource", c.testCreateResource)
}

// E2ETests runs create notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
