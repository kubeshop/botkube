package delete

import (
	"encoding/json"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

func (c *context) testDeleteEvent(t *testing.T) {
	events := []config.EventType{"update", "error", "create"}

	// Modifying AllowedEventKindsMap to remove events other than delete for pod resource
	observedEventKindsMap := c.Ctrl.ObservedEventKindsMap()
	for _, event := range events {
		delete(observedEventKindsMap, controller.EventKind{Resource: "v1/pods", Namespace: "all", EventType: event})
	}
	c.Ctrl.SetObservedEventKindsMap(observedEventKindsMap)

	// test scenarios
	tests := map[string]testutils.DeleteObjects{
		"perform delete operation and configure BotKube to listen only delete events": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "danger", Title: "v1/pods deleted", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-delete* has been deleted in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-delete", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "delete", Level: "critical", Reason: "", Error: ""},
				Summary:     "Pod *test/test-pod-delete* has been deleted in *test-cluster-1* cluster\n",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			isAllowed := c.Ctrl.ShouldSendEvent(test.Namespace, resource, config.DeleteEvent)
			assert.Equal(t, isAllowed, true)

			testutils.DeleteResource(t, c.DynamicCli, test)

			if c.TestEnv.Config.Communications.Slack.Enabled {
				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage()

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				if len(m.Attachments) != 0 {
					m.Attachments[0].Ts = ""
				}
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
		})
	}
	// Resetting original configuration as per test_config
	observedEventKindsMap = c.Ctrl.ObservedEventKindsMap()
	for _, event := range events {
		observedEventKindsMap[controller.EventKind{Resource: "v1/pods", Namespace: "all", EventType: event}] = true
	}
	c.Ctrl.SetObservedEventKindsMap(observedEventKindsMap)
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("delete resource", c.testDeleteEvent)
}

// E2ETests runs delete notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
