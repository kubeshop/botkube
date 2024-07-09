package sink

import (
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
