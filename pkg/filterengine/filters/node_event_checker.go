// NodeEventsChecker filter to send notifications on critical node events

package filters

import (
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	coreV1 "k8s.io/api/core/v1"

	log "github.com/infracloudio/botkube/pkg/logging"
)

const (
	// NodeNotReady EventReason when Node is NotReady
	NodeNotReady string = "NodeNotReady"
	// NodeReady EventReason when Node is Ready
	NodeReady string = "NodeReady"
)

// NodeEventsChecker checks job status and adds message in the events structure
type NodeEventsChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(NodeEventsChecker{
		Description: "Sends notifications on node level critical events.",
	})
}

// Run filers and modifies event struct
func (f NodeEventsChecker) Run(object interface{}, event *events.Event) {

	// Check for Event object
	_, ok := object.(*coreV1.Event)
	if !ok {
		return
	}

	// Run filter only on Node events
	if event.Kind != "Node" {
		return
	}

	// Update event details
	// Promote InfoEvent with critical reason as significant ErrorEvent
	switch event.Reason {
	case NodeNotReady:
		event.Type = config.ErrorEvent
		event.Level = events.Critical
	case NodeReady:
		event.Type = config.InfoEvent
		event.Level = events.Info
	default:
		// skip events with least significant reasons
		event.Skip = true
	}

	log.Logger.Debug("Node Critical Event filter successful!")
}

// Describe filter
func (f NodeEventsChecker) Describe() string {
	return f.Description
}
