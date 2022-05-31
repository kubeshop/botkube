// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package filters

import (
	"encoding/json"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends notification when a resource is created
func (c *context) testFilters(t *testing.T) {
	// Test cases
	tests := map[string]utils.CreateObjects{
		"test ImageTagChecker filter": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nginx-pod", Labels: map[string]string{"env": "test"}}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "nginx", Image: "nginx:latest"}}}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "v1/pods created", Fields: []slack.AttachmentField{{Value: "Pod *test/nginx-pod* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- :latest tag used in image 'nginx:latest' of Container 'nginx' should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "nginx-pod", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Pod *test/nginx-pod* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- :latest tag used in image 'nginx:latest' of Container 'nginx' should be avoided.\n```",
			},
		},

		"test PodLabelChecker filter": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-wo-label"}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "v1/pods created", Fields: []slack.AttachmentField{{Value: "Pod *test/pod-wo-label* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- pod 'pod-wo-label' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "pod-wo-label", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Pod *test/pod-wo-label* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- pod 'pod-wo-label' creation without labels should be avoided.\n```",
			},
		},

		"test IngressValidator filter": {
			GVR:       schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
			Kind:      "Ingress",
			Namespace: "test",
			Specs:     &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ingress-with-service"}, Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Path: "testpath", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test-service", Port: networkingv1.ServiceBackendPort{Number: int32(80)}}}}}}}}}}},
			ExpectedSlackMessage: utils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "networking.k8s.io/v1/ingresses created", Fields: []slack.AttachmentField{{Value: "Ingress *test/ingress-with-service* has been created in *test-cluster-1* cluster\n```\nWarnings:\n- Service 'test-service' used in ingress 'ingress-with-service' config does not exist or port '80' not exposed\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: utils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Ingress", Name: "ingress-with-service", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Ingress *test/ingress-with-service* has been created in *test-cluster-1* cluster\n```\nWarnings:\n- Service 'test-service' used in ingress 'ingress-with-service' config does not exist or port '80' not exposed\n```",
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
