package filters

import (
	"encoding/json"
	"testing"

	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/utils"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends notification when a resource is created
func (c *context) testFilters(t *testing.T) {
	// Test cases
	tests := map[string]utils.CreateObjects{
		"test ImageTagChecker filter": {
			Kind:      "pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nginx-pod", Labels: map[string]string{"env": "test"}}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "nginx", Image: "nginx:latest"}}}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Pod create", Value: "Pod `nginx-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- :latest tag used in image 'nginx:latest' of Container 'nginx' should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "nginx-pod", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: "", Messages: []string{"Resource created\n"}},
				Summary:     "Pod `nginx-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- :latest tag used in image 'nginx:latest' of Container 'nginx' should be avoided.\n```",
			},
		},

		"test PodLabelChecker filter": {
			Kind:      "pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-wo-label"}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Pod create", Value: "Pod `pod-wo-label` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'pod-wo-label' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "pod-wo-label", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: "", Messages: []string{"Resource created\n"}},
				Summary:     "Pod `pod-wo-label` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'pod-wo-label' creation without labels should be avoided.\n```",
			},
		},

		"test IngressValidator filter": {
			Kind:      "ingress",
			Namespace: "test",
			Specs:     &extV1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ingress-with-service"}, Spec: extV1beta1.IngressSpec{Rules: []extV1beta1.IngressRule{{IngressRuleValue: extV1beta1.IngressRuleValue{HTTP: &extV1beta1.HTTPIngressRuleValue{Paths: []extV1beta1.HTTPIngressPath{{Path: "testpath", Backend: extV1beta1.IngressBackend{ServiceName: "test-service", ServicePort: intstr.FromInt(80)}}}}}}}}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Ingress create", Value: "Ingress `ingress-with-service` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nWarnings:\n- Service 'test-service' used in ingress 'ingress-with-service' config does not exist or port '80' not exposed\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Ingress", Name: "ingress-with-service", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: "", Messages: []string{"Resource created\n"}},
				Summary:     "Ingress `ingress-with-service` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nWarnings:\n- Service 'test-service' used in ingress 'ingress-with-service' config does not exist or port '80' not exposed\n```",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Inject an event into the fake client.
			utils.CreateResource(t, test)

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
		})
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("test filters", c.testFilters)
}

// E2ETests runs filter tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
