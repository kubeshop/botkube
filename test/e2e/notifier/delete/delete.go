package delete

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

func (c *context) testSKipDeleteEvent(t *testing.T) {
	// Modifying AllowedEventKindsMap to remove delete event for pod resource
	delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "delete"})
	// test scenarios
	tests := map[string]testutils.DeleteObjects{
		"skip delete event for resources not configured": {
			// delete operation not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if delete operation is skipped
			isAllowed := utils.AllowedEventKindsMap[utils.EventKind{
				Resource:  resource,
				Namespace: "all",
				EventType: config.DeleteEvent}] ||
				utils.AllowedEventKindsMap[utils.EventKind{
					Resource:  resource,
					Namespace: test.Namespace,
					EventType: config.DeleteEvent}]
			assert.Equal(t, isAllowed, false)
		})
	}
	// Resetting original configuration as per test_config
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "delete"}] = true
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("delete resource", c.testSKipDeleteEvent)
}

// E2ETests runs delete notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
