package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	metricPrefix = "botkube_"
)

// Reporter provides methods for updating Botkube metrics.
type Reporter interface {
	UpdateConfigurationMetrics(cfg config.Config)
	ReportEventEvaluated(source string)
	ReportCommandExecuted(executor string)
}

type prometheusReporter struct {
	configuredCommunications *prometheus.GaugeVec
	configuredSources        *prometheus.GaugeVec
	configuredExecutors      *prometheus.GaugeVec

	totalEvaluatedEvents     prometheus.Counter
	sourceEvaluatedEvents    *prometheus.CounterVec
	totalExecutedCommands    prometheus.Counter
	executorExecutedCommands *prometheus.CounterVec
}

// NewPrometheusReporter creates a new Reporter that registers Botkube-specific Prometheus metrics.
func NewPrometheusReporter() Reporter {
	return &prometheusReporter{
		configuredCommunications: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_communications",
			Help: "Number of configured communications by Botkube",
		}, []string{"id"}),
		configuredSources: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_sources",
			Help: "Number of configured sources by Botkube",
		}, []string{"id"}),
		configuredExecutors: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_executors",
			Help: "Number of configured executors by Botkube",
		}, []string{"id"}),

		totalEvaluatedEvents: promauto.NewCounter(prometheus.CounterOpts{
			Name: metricPrefix + "total_evaluated_events_count",
			Help: "Total number of evaluated events",
		}),
		sourceEvaluatedEvents: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: metricPrefix + "source_evaluated_events_count",
			Help: "Number of evaluated events by source",
		}, []string{"source"}),
		totalExecutedCommands: promauto.NewCounter(prometheus.CounterOpts{
			Name: metricPrefix + "total_executed_commands_count",
			Help: "Total number of executed commands",
		}),
		executorExecutedCommands: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: metricPrefix + "executor_executed_commands_count",
			Help: "Number of executed commands by executor",
		}, []string{"executor"}),
	}
}

// UpdateConfigurationMetrics updates the configuration-related metrics based on the provided Config.
func (r *prometheusReporter) UpdateConfigurationMetrics(cfg config.Config) {
	// Reset gauges before setting to handle removed configurations
	r.configuredCommunications.Reset()
	r.configuredSources.Reset()
	r.configuredExecutors.Reset()

	for commName := range cfg.Communications {
		r.configuredCommunications.WithLabelValues(commName).Set(1)
	}

	for sourceName := range cfg.Sources {
		r.configuredSources.WithLabelValues(sourceName).Set(1)
	}

	for execName := range cfg.Executors {
		r.configuredExecutors.WithLabelValues(execName).Set(1)
	}
}

// ReportEventEvaluated increments the counters for evaluated events.
func (r *prometheusReporter) ReportEventEvaluated(source string) {
	r.totalEvaluatedEvents.Inc()
	r.sourceEvaluatedEvents.WithLabelValues(source).Inc()
}

// ReportCommandExecuted increments the counters for executed commands.
func (r *prometheusReporter) ReportCommandExecuted(executor string) {
	r.totalExecutedCommands.Inc()
	r.executorExecutedCommands.WithLabelValues(executor).Inc()
}

// NoopReporter is a no-op implementation of Reporter.
type NoopReporter struct{}

// NewNoopReporter creates a new NoopReporter.
func NewNoopReporter() Reporter {
	return &NoopReporter{}
}

func (n *NoopReporter) UpdateConfigurationMetrics(cfg config.Config) {}
func (n *NoopReporter) ReportEventEvaluated(source string)           {}
func (n *NoopReporter) ReportCommandExecuted(executor string)        {}
