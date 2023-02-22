package audit

import (
	"context"

	"github.com/sirupsen/logrus"
)

var _ AuditReporter = (*NoopAuditReporter)(nil)

type NoopAuditReporter struct {
	log logrus.FieldLogger
}

func newNoopAuditReporter(logger logrus.FieldLogger) *NoopAuditReporter {
	return &NoopAuditReporter{
		log: logger,
	}
}
func (r *NoopAuditReporter) ReportExecutorAuditEvent(ctx context.Context, e AuditEvent) error {
	r.log.Debug("ReportExecutorAuditEvent")
	return nil
}

func (r *NoopAuditReporter) ReportSourceAuditEvent(ctx context.Context, e AuditEvent) error {
	r.log.Debug("ReportSourceAuditEvent")
	return nil
}
