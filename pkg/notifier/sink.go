package notifier

import (
	"context"

	"github.com/kubeshop/botkube/pkg/config"
)

// Sink sends event notifications to the sinks.
type Sink interface {
	// SendMessage sends a generic message for a given source bindings.
	SendMessage(context.Context, any, []string) error

	// IntegrationName returns a name of a given communication platform.
	IntegrationName() config.CommPlatformIntegration

	// Type returns a given integration type. See config.IntegrationType for possible integration types.
	Type() config.IntegrationType
}
