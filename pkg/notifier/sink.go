package notifier

import (
	"context"

	"github.com/kubeshop/botkube/pkg/config"
)

type Status struct {
	Status   StatusMsg
	Restarts string
	Reason   FailureReasonMsg
}

type StatusMsg string
type FailureReasonMsg string

const (
	StatusUnknown   StatusMsg = "Unknown"
	StatusHealthy   StatusMsg = "Healthy"
	StatusUnHealthy StatusMsg = "Unhealthy"
)

const (
	FailureReasonConnectionError FailureReasonMsg = "Connection error"
)

// Sink sends event notifications to the sinks.
type Sink interface {
	// SendEvent sends a generic event for a given source bindings.
	SendEvent(context.Context, any, []string) error

	// IntegrationName returns a name of a given communication platform.
	IntegrationName() config.CommPlatformIntegration

	// Type returns a given integration type. See config.IntegrationType for possible integration types.
	Type() config.IntegrationType

	// GetStatus gets sink status
	GetStatus() Status
}
