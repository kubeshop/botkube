package sink

import (
	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Sink sends messages to communication channels. It is a one-way integration.
type Sink interface {
	notifier.Sink
}

// AnalyticsReporter defines a reporter that collects analytics data for sinks.
type AnalyticsReporter interface {
	// ReportSinkEnabled reports an enabled sink.
	ReportSinkEnabled(platform config.CommPlatformIntegration, commGroupIdx int) error
}

type HealthNotifierSink interface {
	GetStatus() health.PlatformStatus
}

// FailedSink mock of sink, uses for healthChecker.
type FailedSink struct {
	status        health.PlatformStatusMsg
	failureReason health.FailureReasonMsg
	errorMsg      string
}

// NewSinkFailed creates a new FailedSink instance.
func NewSinkFailed(failureReason health.FailureReasonMsg, errorMsg string) *FailedSink {
	return &FailedSink{
		status:        health.StatusUnHealthy,
		failureReason: failureReason,
		errorMsg:      errorMsg,
	}
}

// GetStatus gets bot status.
func (s *FailedSink) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   s.status,
		Restarts: "0/0",
		Reason:   s.failureReason,
		ErrorMsg: s.errorMsg,
	}
}
