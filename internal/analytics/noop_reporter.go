package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

var _ Reporter = &NoopReporter{}

// NoopReporter implements Reporter interface, but is a no-op (doesn't execute any logic).
type NoopReporter struct{}

// NewNoopReporter creates new NoopReporter instance.
func NewNoopReporter() *NoopReporter {
	return &NoopReporter{}
}

// RegisterCurrentIdentity loads the current anonymous identity and registers it.
func (n NoopReporter) RegisterCurrentIdentity(_ context.Context, _ kubernetes.Interface, _ string) error {
	return nil
}
