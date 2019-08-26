package utils

import (
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/nlopes/slack"
	"k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SlackMessage structure
type SlackMessage struct {
	Text        string
	Attachments []slack.Attachment
}

// CreateObjects stores specs for creating a k8s fake object and expected Slack response
type CreateObjects struct {
	Kind      string
	Namespace string
	Specs     runtime.Object
	NotifType config.NotifType
	Expected  SlackMessage
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
		s := obj.Specs.(*extV1beta1.Ingress)
		_, err := utils.KubeClient.ExtensionsV1beta1().Ingresses(obj.Namespace).Create(s)
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	}
}
