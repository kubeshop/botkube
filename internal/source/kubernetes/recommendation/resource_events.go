package recommendation

import (
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/pkg/ptr"
)

const (
	podsResourceType    = "v1/pods"
	ingressResourceType = "networking.k8s.io/v1/ingresses"
)

// ResourceEventsForConfig returns the resource event map for a given source recommendations config.
func ResourceEventsForConfig(recCfg *config.Recommendations) map[string]config.EventType {
	resTypes := make(map[string]config.EventType)
	if recCfg == nil {
		return resTypes
	}

	if ptr.IsTrue(recCfg.Ingress.TLSSecretValid) || ptr.IsTrue(recCfg.Ingress.BackendServiceValid) {
		resTypes[ingressResourceType] = config.CreateEvent
	}

	if ptr.IsTrue(recCfg.Pod.NoLatestImageTag) || ptr.IsTrue(recCfg.Pod.LabelsSet) {
		resTypes[podsResourceType] = config.CreateEvent
	}

	return resTypes
}

// ShouldIgnoreEvent returns true if user doesn't listen to events for a given resource, apart from enabled recommendations.
func ShouldIgnoreEvent(recCfg *config.Recommendations, event event.Event) bool {
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
