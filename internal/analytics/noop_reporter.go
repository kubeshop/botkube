package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

var _ Reporter = &NoopReporter{}

type NoopReporter struct{}

func NewNoopReporter() *NoopReporter {
	return &NoopReporter{}
}

func (n NoopReporter) RegisterCurrentIdentity(_ context.Context, _ kubernetes.Interface, _ string) error {
	return nil
}
