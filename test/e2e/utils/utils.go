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

package utils

import (
	"strings"
	"testing"

	"github.com/nlopes/slack"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
)

var (
	podGVR       = utils.ParseResourceArg("v1/pods")
	serviceGVR   = utils.ParseResourceArg("v1/services")
	ingressGVR   = utils.ParseResourceArg("networking.k8s.io/v1beta1/ingresses")
	namespaceGVR = utils.ParseResourceArg("v1/namespaces")
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
		// convert the runtime.Object to unstructured.Unstructured
		s := unstructured.Unstructured{}
		k, ok := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
		if ok != nil {
			t.Fatalf("Failed to convert pod object into unstructured")
		}
		s.Object = k
		s.SetGroupVersionKind(podGVR.GroupVersion().WithKind(strings.Title(strings.ToLower(obj.Kind))))
		_, err := utils.DynamicKubeClient.Resource(podGVR).Namespace(obj.Namespace).Create(&s, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
	case "service":
		s := unstructured.Unstructured{}
		k, ok := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
		if ok != nil {
			t.Fatalf("Failed to convert pod object into unstructured")
		}
		s.Object = k
		s.SetGroupVersionKind(serviceGVR.GroupVersion().WithKind(strings.Title(strings.ToLower(obj.Kind))))
		_, err := utils.DynamicKubeClient.Resource(serviceGVR).Namespace(obj.Namespace).Create(&s, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	case "ingress":
		s := unstructured.Unstructured{}
		k, ok := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
		if ok != nil {
			t.Fatalf("Failed to convert pod object into unstructured")
		}
		s.Object = k
		s.SetGroupVersionKind(ingressGVR.GroupVersion().WithKind(strings.Title(strings.ToLower(obj.Kind))))

		_, err := utils.DynamicKubeClient.Resource(ingressGVR).Namespace(obj.Namespace).Create(&s, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create ingress: %v", err)
		}
	case "namespace":
		s := unstructured.Unstructured{}
		k, ok := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
		if ok != nil {
			t.Fatalf("Failed to convert pod object into unstructured")
		}
		s.Object = k
		s.SetGroupVersionKind(namespaceGVR.GroupVersion().WithKind(strings.Title(strings.ToLower(obj.Kind))))
		_, err := utils.DynamicKubeClient.Resource(namespaceGVR).Namespace(obj.Namespace).Create(&s, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create namespace: %v", err)
		}

	default:
		t.Fatalf("CreateResource method is not defined for resource %s", obj.Kind)
	}
}
