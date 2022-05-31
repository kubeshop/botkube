package error

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

func (c *context) testSKipErrorEvent(t *testing.T) {
	// Modifying AllowedEventKindsMap to add error event for only dummy namespace and ignore everything
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "error"}] = true

	serviceEvent := utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/services", Namespace: "all", EventType: "error"}]
	podEvent := utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "error"}]

	// Modifying AllowedEventKindsMap to remove error event for pod resource
	delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "error"})

	// Modifying AllowedEventKindsMap to remove error event for service resource
	delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/services", Namespace: "all", EventType: "error"})

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
			// checking if error operation is skipped
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.ErrorEvent)
			assert.Equal(t, isAllowed, false)
		})
	}
	// Resetting original configuration as per test_config
	defer func() {
		utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/services", Namespace: "all", EventType: "error"}] = serviceEvent
		utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "error"}] = podEvent
		delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "error"})
	}()
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("skip error event", c.testSKipErrorEvent)
}

// E2ETests runs error notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
