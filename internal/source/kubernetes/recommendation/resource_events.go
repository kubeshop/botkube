package recommendation

import (
	config2 "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/pkg/ptr"
)

const (
	podsResourceType    = "v1/pods"
	ingressResourceType = "networking.k8s.io/v1/ingresses"
)

// ResourceEventsForConfig returns the resource event map for a given source recommendations config.
func ResourceEventsForConfig(recCfg config2.Recommendations) map[string]config2.EventType {
	resTypes := make(map[string]config2.EventType)

	if ptr.IsTrue(recCfg.Ingress.TLSSecretValid) || ptr.IsTrue(recCfg.Ingress.BackendServiceValid) {
		resTypes[ingressResourceType] = config2.CreateEvent
	}

	if ptr.IsTrue(recCfg.Pod.NoLatestImageTag) || ptr.IsTrue(recCfg.Pod.LabelsSet) {
		resTypes[podsResourceType] = config2.CreateEvent
	}

	return resTypes
}

// ShouldIgnoreEvent returns true if user doesn't listen to events for a given resource, apart from enabled recommendations.
func ShouldIgnoreEvent(recCfg config2.Recommendations, event event.Event) bool {
	if event.HasRecommendationsOrWarnings() {
		// shouldn't be skipped
		return false
	}

	res := ResourceEventsForConfig(recCfg)
	recommEventType, ok := res[event.Resource]
	if !ok {
		// this event doesn't relate to recommendations, finish early
		return false
	}

	if event.Type != recommEventType {
		// this event doesn't relate to recommendations, finish early
		return false
	}

	// The event is related to recommendations informers. No recommendations there, so it should be skipped.
	return true
}
