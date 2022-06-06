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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"

	"github.com/slack-go/slack"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
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
	GVR                    schema.GroupVersionResource
	Kind                   string
	Namespace              string
	Specs                  runtime.Object
	NotifType              config.NotifType
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// UpdateObjects stores specs and patch for updating a k8s fake object and expected Slack response
type UpdateObjects struct {
	GVR                    schema.GroupVersionResource
	Kind                   string
	Namespace              string
	Name                   string
	Specs                  runtime.Object
	Patch                  []byte
	Diff                   string
	UpdateSetting          config.UpdateSetting
	NotifType              config.NotifType
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// DeleteObjects stores specs for deleting a k8s fake object
type DeleteObjects struct {
	GVR                    schema.GroupVersionResource
	Kind                   string
	Namespace              string
	Name                   string
	Specs                  runtime.Object
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// ErrorEvent stores specs for throwing an error in case of anomalies
type ErrorEvent struct {
	GVR                    schema.GroupVersionResource
	Kind                   string
	Namespace              string
	Name                   string
	Specs                  runtime.Object
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// CreateResource with fake client
func CreateResource(t *testing.T, dynamicCli dynamic.Interface, obj CreateObjects) {
	// convert the runtime.Object to unstructured.Unstructured
	s := unstructured.Unstructured{}
	k, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
	require.NoError(t, err, "while converting pod object into unstructured")

	s.Object = k
	s.SetGroupVersionKind(obj.GVR.GroupVersion().WithKind(obj.Kind))
	// Create resource
	_, err = dynamicCli.Resource(obj.GVR).Namespace(obj.Namespace).Create(context.TODO(), &s, v1.CreateOptions{})
	require.NoError(t, err, "while creating %q", obj.GVR.Resource)
}

// UpdateResource Create and update the obj and return old and new obj
func UpdateResource(t *testing.T, dynamicCli dynamic.Interface, obj UpdateObjects) (*unstructured.Unstructured, *unstructured.Unstructured) {
	s := unstructured.Unstructured{}
	k, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
	require.NoError(t, err, "while converting pod object into unstructured")

	s.Object = k
	s.SetGroupVersionKind(obj.GVR.GroupVersion().WithKind(obj.Kind))
	// Create resource and get the old object
	oldObj, err := dynamicCli.Resource(obj.GVR).Namespace(obj.Namespace).Create(context.TODO(), &s, v1.CreateOptions{})
	require.NoError(t, err, "while creating %q", obj.GVR.Resource)

	// Mock the time delay involved in listening, filtering, and notifying events to all notifiers
	time.Sleep(10 * time.Second) // TODO: Is that really needed? Improve it in https://github.com/infracloudio/botkube/issues/589
	// Applying patch
	newObj, err := dynamicCli.Resource(obj.GVR).Namespace(obj.Namespace).Patch(context.TODO(), s.GetName(), types.MergePatchType, obj.Patch, v1.PatchOptions{})
	require.NoError(t, err, "while updating %q", obj.GVR.Resource)

	// Mock the time delay involved in listening, filtering, and notifying events to all notifiers
	time.Sleep(10 * time.Second) // TODO: Is that really needed? Improve it in https://github.com/infracloudio/botkube/issues/589
	return oldObj, newObj
}

// DeleteResource deletes the obj with fake client
func DeleteResource(t *testing.T, dynamicCli dynamic.Interface, obj DeleteObjects) {
	s := unstructured.Unstructured{}
	k, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.Specs)
	require.NoError(t, err, "while converting pod object into unstructured")

	s.Object = k
	s.SetGroupVersionKind(obj.GVR.GroupVersion().WithKind(obj.Kind))

	_, err = dynamicCli.Resource(obj.GVR).Namespace(obj.Namespace).Create(context.TODO(), &s, v1.CreateOptions{})
	require.NoError(t, err, "while creating %q", obj.GVR.Resource)

	// Delete resource
	err = dynamicCli.Resource(obj.GVR).Namespace(obj.Namespace).Delete(context.TODO(), s.GetName(), v1.DeleteOptions{})
	require.NoError(t, err, "while deleting %q", obj.GVR.Resource)
}
