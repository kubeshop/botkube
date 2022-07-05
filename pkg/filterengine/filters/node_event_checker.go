package filters

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	// NodeNotReady EventReason when Node is NotReady
	NodeNotReady string = "NodeNotReady"
	// NodeReady EventReason when Node is Ready
	NodeReady string = "NodeReady"
)

// NodeEventsChecker checks job status and adds message in the events structure
type NodeEventsChecker struct {
	log logrus.FieldLogger
}

// NewNodeEventsChecker creates a new NodeEventsChecker instance
func NewNodeEventsChecker(log logrus.FieldLogger) *NodeEventsChecker {
	return &NodeEventsChecker{log: log}
}

// Run filers and modifies event struct
func (f *NodeEventsChecker) Run(_ context.Context, object interface{}, event *events.Event) error {
	// Check for Event object
	if utils.GetObjectTypeMetaData(object).Kind == "Event" {
		return nil
	}

	// Run filter only on Node events
	if event.Kind != "Node" {
		return nil
	}

	// Update event details
	// Promote InfoEvent with critical reason as significant ErrorEvent
	switch event.Reason {
	case NodeNotReady:
		event.Type = config.ErrorEvent
		event.Level = config.Critical
	case NodeReady:
		event.Type = config.InfoEvent
		event.Level = config.Info
	default:
		// skip events with least significant reasons
		event.Skip = true
	}

	f.log.Debug("Node Critical Event filter successful!")
	return nil
}

// Name returns the filter's name
func (f *NodeEventsChecker) Name() string {
	return "NodeEventsChecker"
}

// Describe describes the filter
func (f *NodeEventsChecker) Describe() string {
	return "Sends notifications on node level critical events."
}
