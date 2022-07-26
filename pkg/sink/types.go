package sink

import (
	"github.com/kubeshop/botkube/pkg/config"
)

// AnalyticsReporter defines a reporter that collects analytics data for sinks.
type AnalyticsReporter interface {
	// ReportSinkEnabled reports an enabled sink.
	ReportSinkEnabled(platform config.CommPlatformIntegration) error
}
