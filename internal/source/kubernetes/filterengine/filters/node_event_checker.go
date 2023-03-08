package filters

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/k8sutil"
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
func (f *NodeEventsChecker) Run(_ context.Context, event *event.Event) error {
	// Check for Event object
	if k8sutil.GetObjectTypeMetaData(event.Object).Kind == "Event" {
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
		event.Level = config.Error
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
