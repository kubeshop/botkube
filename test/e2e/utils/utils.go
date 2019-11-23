package utils

import (
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/nlopes/slack"
	v1 "k8s.io/api/core/v1"
	networkV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SlackMessage structure
type SlackMessage struct {
	Text        string
	Attachments []slack.Attachment
}

// WebhookPayload structure
type WebhookPayload struct {
	Summary     string             `json:"summary"`
	EventMeta   notify.EventMeta   `json:"meta"`
	EventStatus notify.EventStatus `json:"status"`
}

// CreateObjects stores specs for creating a k8s fake object and expected Slack response
type CreateObjects struct {
	Kind                   string
	Namespace              string
	Specs                  runtime.Object
	NotifType              config.NotifType
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// CreateResource with fake client
func CreateResource(t *testing.T, obj CreateObjects) {
	switch obj.Kind {
	case "pod":
		s := obj.Specs.(*v1.Pod)
		_, err := utils.KubeClient.CoreV1().Pods(obj.Namespace).Create(s)
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
	case "service":
		s := obj.Specs.(*v1.Service)
		_, err := utils.KubeClient.CoreV1().Services(obj.Namespace).Create(s)
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	case "ingress":
		s := obj.Specs.(*networkV1beta1.Ingress)
		_, err := utils.KubeClient.NetworkingV1beta1().Ingresses(obj.Namespace).Create(s)
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	case "namespace":
		s := obj.Specs.(*v1.Namespace)
		_, err := utils.KubeClient.CoreV1().Namespaces().Create(s)
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	default:
		t.Fatalf("CreateResource method is not defined for resource %s", obj.Kind)
	}
}
