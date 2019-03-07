package filters

import (
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"

	apiV1 "k8s.io/api/core/v1"
)

// PodLabelChecker add recommendations to the event object if pod created without any labels
type PodLabelChecker struct {
}

// NewPodLabelChecker creates new PodLabelChecker object
func NewPodLabelChecker() *PodLabelChecker {
	return &PodLabelChecker{}
}

// Run filters and modifies event struct
func (f *PodLabelChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" && event.Type != "create" {
		return
	}
	podObj, ok := object.(*apiV1.Pod)
	if !ok {
		return
	}

	// Check labels in pod
	if len(podObj.ObjectMeta.Labels) == 0 {
		event.Recommendations = append(event.Recommendations, "pod '"+podObj.ObjectMeta.Name+"' creation without labels should be avoided.\n")
	}
	log.Logger.Debug("Pod label filter successful!")
}
