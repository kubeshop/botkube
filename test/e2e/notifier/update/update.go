package update

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/infracloudio/botkube/pkg/controller"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

func (c *context) testUpdateResource(t *testing.T) {
	// Test cases
	tests := map[string]testutils.UpdateObjects{
		"update resource when IncludeDiff is set to false": {
			// Diff message should not be generated in Attachment if IncludeDiff field is false
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update-diff-false"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "warning", Title: "v1/pods updated", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-update-diff-false* has been updated in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update-diff-false",
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
			UpdateSetting: config.UpdateSetting{Fields: []string{"spec.containers[*].image"}, IncludeDiff: false},
			Diff:          "spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n",
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-update-diff-false", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "update", Level: "warn", Reason: "", Error: "", Messages: []string(nil)},
				Summary:     "Pod *test/test-pod-update-diff-false* has been updated in *test-cluster-1* cluster\n",
			},
		},
		"create and update pod in configured namespace": {
			// Diff message generated in Attachment if IncludeDiff field is true
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
			UpdateSetting: config.UpdateSetting{Fields: []string{"spec.containers[*].image"}, IncludeDiff: true},
			Diff:          "spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n",
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-update", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "update", Level: "warn", Reason: "", Error: "", Messages: []string{"spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n"}},
				Summary:     "Pod *test/test-pod-update* has been updated in *test-cluster-1* cluster\n```\nspec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n\n```",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if update operation is true
			observedEventKindsMap := c.Ctrl.ObservedEventKindsMap()
			isAllowed := observedEventKindsMap[controller.EventKind{
				Resource:  resource,
				Namespace: "all",
				EventType: config.UpdateEvent}] ||
				observedEventKindsMap[controller.EventKind{
					Resource:  resource,
					Namespace: test.Namespace,
					EventType: config.UpdateEvent}]
			assert.Equal(t, isAllowed, true)
			// modifying the update setting value as per testcases
			observedUpdateEventKindsMap := c.Ctrl.ObservedUpdateEventsMap()
			observedUpdateEventKindsMap[controller.KindNS{Resource: "v1/pods", Namespace: "all"}] = test.UpdateSetting
			c.Ctrl.SetObservedUpdateEventsMap(observedUpdateEventKindsMap)
			// getting the updated and old object
			oldObj, newObj := testutils.UpdateResource(t, c.DynamicCli, test)
			updateMsg, err := utils.Diff(oldObj.Object, newObj.Object, test.UpdateSetting)
			require.NoError(t, err)
			assert.Equal(t, test.Diff, updateMsg)
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
				t.Logf("LastSeenPayload :%#v", lastSeenPayload)
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
	t.Run("skip update event for wrong setting", c.testSkipWrongSetting)
}

// E2ETests runs create notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}

func (c *context) testSkipWrongSetting(t *testing.T) {
	// test scenarios
	tests := map[string]struct {
		updateObj          testutils.UpdateObjects
		expectedErrMessage string
	}{
		"skip update event for wrong updateSettings value": {
			updateObj: testutils.UpdateObjects{
				// update event given with wrong value of updateSettings which doesn't exist would be skipped
				GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
				Kind:      "Pod",
				Namespace: "test",
				Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update-skip"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
				Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update-skip",
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
				// adding wrong field
				UpdateSetting: config.UpdateSetting{Fields: []string{"spec.invalid"}, IncludeDiff: true},
				// diff calcuted should be empty because of error
				Diff: "",
			},
			expectedErrMessage: "while finding value from jsonpath: \"spec.invalid\", object: map[apiVersion:v1 kind:Pod metadata:map[creationTimestamp:<nil> name:test-pod-update-skip namespace:test] spec:map[containers:[map[image:tomcat:9.0.34 name:test-pod-container resources:map[]]]] status:map[]]: invalid is not found",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.updateObj.GVR)
			// checking if update operation is true
			isAllowed := c.Ctrl.ShouldSendEvent(test.updateObj.Namespace, resource, config.UpdateEvent)
			assert.Equal(t, isAllowed, true)
			// modifying the update setting value as per testcases

			observedUpdateEventKindsMap := c.Ctrl.ObservedUpdateEventsMap()
			observedUpdateEventKindsMap[controller.KindNS{Resource: "v1/pods", Namespace: "all"}] = test.updateObj.UpdateSetting
			c.Ctrl.SetObservedUpdateEventsMap(observedUpdateEventKindsMap)

			// getting the updated and old object
			oldObj, newObj := testutils.UpdateResource(t, c.DynamicCli, test.updateObj)
			updateMsg, err := utils.Diff(oldObj.Object, newObj.Object, test.updateObj.UpdateSetting)
			if test.expectedErrMessage != "" {
				require.EqualError(t, err, test.expectedErrMessage)
			}

			assert.Equal(t, test.updateObj.Diff, updateMsg)
		})
	}
}
