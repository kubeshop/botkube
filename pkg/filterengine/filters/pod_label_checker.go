package filters

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"

	coreV1 "k8s.io/api/core/v1"
)

// PodLabelChecker add recommendations to the event object if pod created without any labels
type PodLabelChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(PodLabelChecker{
		Description: "Checks and adds recommedations if labels are missing in the pod specs.",
	})
}

// Run filters and modifies event struct
func (f PodLabelChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" && event.Type != config.CreateEvent {
		return
	}
	podObj, ok := object.(*coreV1.Pod)
	if !ok {
		return
	}

	// Check labels in pod
	if len(podObj.ObjectMeta.Labels) == 0 {
		event.Recommendations = append(event.Recommendations, fmt.Sprintf("pod '%s' creation without labels should be avoided.", podObj.ObjectMeta.Name))
	}
	log.Logger.Debug("Pod label filter successful!")
}

// Describe filter
func (f PodLabelChecker) Describe() string {
	return f.Description
}
