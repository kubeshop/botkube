package status

import (
	"context"
)

var _ StatusReporter = (*NoopStatusReporter)(nil)

type NoopStatusReporter struct {}

func newNoopStatusReporter() *NoopStatusReporter {
	return &NoopStatusReporter{}
}
func (r *NoopStatusReporter) ReportDeploymentStartup(context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) ReportDeploymentShutdown(context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) ReportDeploymentFailed(context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) SetResourceVersion(int) { }
