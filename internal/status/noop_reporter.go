package status

import (
	"context"

	"github.com/sirupsen/logrus"
)

var _ Reporter = (*NoopStatusReporter)(nil)

type NoopStatusReporter struct{}

func (n NoopStatusReporter) ReportDeploymentConnectionInit(context.Context, string) error {
	return nil
}

func (n NoopStatusReporter) ReportDeploymentStartup(context.Context) error {
	return nil
}

func (n NoopStatusReporter) AckNewResourceVersion(context.Context) error {
	return nil
}

func (n NoopStatusReporter) ReportDeploymentShutdown(context.Context) error {
	return nil
}

func (n NoopStatusReporter) ReportDeploymentFailure(context.Context, string) error {
	return nil
}

func (n NoopStatusReporter) SetResourceVersion(int) {
}

func (n NoopStatusReporter) SetLogger(logrus.FieldLogger) {}

func newNoopStatusReporter() *NoopStatusReporter {
	return &NoopStatusReporter{}
}
