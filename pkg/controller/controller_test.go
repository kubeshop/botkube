package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

// TODO: Tests moved out straight from E2E test package with minimal changes.
// 	Refactor them as a part of https://github.com/infracloudio/botkube/issues/589

func TestController_ShouldSendEvent_SkipError(t *testing.T) {
	observedEventKindsMap := map[EventKind]bool{
		{Resource: "v1/pods", Namespace: "dummy", EventType: "error"}: true,
	}

	tests := map[string]testutils.ErrorEvent{
		"skip error event for resources not configured": {
			// error event not allowed for Pod resources so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip error event for namespace not configured": {
			// error event not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip error event for resources not added in test_config": {
			// error event should not be allowed for service resource so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Kind:      "Service",
			Namespace: "test",
			Specs:     &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service-error"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)

			c := Controller{
				observedEventKindsMap: observedEventKindsMap,
			}

			isAllowed := c.shouldSendEvent(test.Namespace, resource, config.ErrorEvent)
			assert.Equal(t, false, isAllowed)
		})
	}
}

func TestController_ShouldSendEvent_SkipUpdate(t *testing.T) {
	observedEventKindsMap := map[EventKind]bool{
		{Resource: "v1/pods", Namespace: "dummy", EventType: "delete"}: true,
	}

	// test scenarios
	tests := map[string]testutils.UpdateObjects{
		"skip update event for namespaces not configured": {
			// update operation not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip update event for resources not added": {
			// update operation not allowed for namespaces in test_config so event should be skipped
			GVR:   schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
			Kind:  "Namespace",
			Specs: &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "abc"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)

			c := Controller{
				observedEventKindsMap: observedEventKindsMap,
			}

			isAllowed := c.shouldSendEvent(test.Namespace, resource, config.ErrorEvent)
			assert.Equal(t, false, isAllowed)
		})
	}
}

func TestController_ShouldSendEvent_SkipDelete(t *testing.T) {
	observedEventKindsMap := map[EventKind]bool{
		{Resource: "v1/pods", Namespace: "dummy", EventType: "delete"}: true,
	}

	// test scenarios
	tests := map[string]testutils.DeleteObjects{
		"skip delete event for resources not configured": {
			// delete operation not allowed for Pod resources so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip delete event for namespace not configured": {
			// delete operation not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip delete event for resources not added in test_config": {
			// delete operation not allowed for Pod resources so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Kind:      "Service",
			Namespace: "test",
			Specs:     &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service-delete"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)

			c := Controller{
				observedEventKindsMap: observedEventKindsMap,
			}

			isAllowed := c.shouldSendEvent(test.Namespace, resource, config.ErrorEvent)
			assert.Equal(t, false, isAllowed)
		})
	}
}
