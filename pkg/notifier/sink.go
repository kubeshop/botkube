package notifier

import (
	"context"

	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/config"
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
	GetStatus() health.PlatformStatus
}
