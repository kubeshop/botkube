package status

import (
	"context"
)

var _ StatusReporter = (*NoopStatusReporter)(nil)

type NoopStatusReporter struct{}

func newNoopStatusReporter() *NoopStatusReporter {
	return &NoopStatusReporter{}
}
func (r *NoopStatusReporter) ReportDeploymentStartup(context.Context) error {
	return nil
}

func (r *NoopStatusReporter) ReportDeploymentShutdown(context.Context) error {
	return nil
}

func (r *NoopStatusReporter) ReportDeploymentFailed(context.Context) error {
	return nil
}

func (r *NoopStatusReporter) SetResourceVersion(int) {}
