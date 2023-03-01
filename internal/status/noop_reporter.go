package status

import (
	"context"

	"github.com/sirupsen/logrus"
)

var _ StatusReporter = (*NoopStatusReporter)(nil)

type NoopStatusReporter struct {
	log logrus.FieldLogger
}

func newNoopStatusReporter(logger logrus.FieldLogger) *NoopStatusReporter {
	return &NoopStatusReporter{
		log: logger,
	}
}
func (r *NoopStatusReporter) ReportDeploymentStartup(ctx context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) ReportDeploymentShutdown(ctx context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) ReportDeploymentFailed(ctx context.Context) (bool, error) {
	return true, nil
}

func (r *NoopStatusReporter) SetResourceVersion(resourceVersion int) {

}
