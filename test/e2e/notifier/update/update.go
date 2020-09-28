package update

import (
	"encoding/json"
	"fmt"
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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type context struct {
	*env.TestEnv
}

func (c *context) testUpdateResource(t *testing.T) {

	// Test cases
	tests := map[string]testutils.UpdateObjects{
		"create and update pod in configured namespace": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "warning", Title: "v1/pods updated", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-update* has been updated in *test-cluster-1* cluster\n```\nspec.containers[*].image:\n\t-: tomcat:9.0.34\n\t : tomcat:8.0\n\n```", Short: false}}, Footer: "BotKube"}},
			},
			Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update",
				  "namespace": "test"
				},
				"spec": {
				  "containers": [
					{
					  "name": "test-pod-container",
					  "image": "tomcat:8.0"
					}
				  ]
				}
			  }
			  `),
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-update", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "update", Level: "warn", Reason: "", Error: "", Messages: []string{"spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n"}},
				Summary:     "Pod *test/test-pod-update* has been updated in *test-cluster-1* cluster\n```\nspec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n\n```",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			//checking if update operation is true
			resource := fmt.Sprintf("%s/%s/%s", test.GVR.Group, test.GVR.Version, test.GVR.Resource)
			if test.GVR.Group == "" {
				resource = fmt.Sprintf("%s/%s", test.GVR.Version, test.GVR.Resource)
			}
			isAllowed := utils.AllowedEventKindsMap[utils.EventKind{
				Resource:  resource,
				Namespace: "all",
				EventType: config.UpdateEvent}] ||
				utils.AllowedEventKindsMap[utils.EventKind{
					Resource:  resource,
					Namespace: test.Namespace,
					EventType: config.UpdateEvent}]
			assert.Equal(t, isAllowed, true)
			// getting the updated and old object
			oldObj, newObj := testutils.UpdateResource(t, test)
			// update setting available
			updateSetting, exist := utils.AllowedUpdateEventsMap[utils.KindNS{Resource: resource, Namespace: "all"}]
			if !exist {
				// Check if specified namespace is allowed
				updateSetting, exist = utils.AllowedUpdateEventsMap[utils.KindNS{Resource: resource, Namespace: test.Namespace}]
			}
			//getting the diff
			updateMsg := utils.Diff(oldObj.Object, newObj.Object, updateSetting)
			assert.Equal(t, "spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n", updateMsg)
			// Inject an event into the fake client.
			if c.TestEnv.Config.Communications.Slack.Enabled {
				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage()

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.Equal(t, c.Config.Communications.Slack.Channel, m.Channel)
				if len(m.Attachments) != 0 {
					m.Attachments[0].Ts = ""
				}
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
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("update resource", c.testUpdateResource)
}

// E2ETests runs create notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
