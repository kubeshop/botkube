// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// NodeEventsChecker filter to send notifications on critical node events

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
