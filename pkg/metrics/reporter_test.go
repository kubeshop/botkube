package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestPrometheusReporter_UpdateConfigurationMetrics(t *testing.T) {
	// We need to create a custom registry to avoid panic on re-registering metrics during tests
	registry := prometheus.NewRegistry()
	
	reporter := &prometheusReporter{
		configuredCommunications: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_communications",
			Help: "Number of configured communications by Botkube",
		}, []string{"id"}),
		configuredSources: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_sources",
			Help: "Number of configured sources by Botkube",
		}, []string{"id"}),
		configuredExecutors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricPrefix + "configured_executors",
			Help: "Number of configured executors by Botkube",
		}, []string{"id"}),
	}
	
	registry.MustRegister(reporter.configuredCommunications)
	registry.MustRegister(reporter.configuredSources)
	registry.MustRegister(reporter.configuredExecutors)

	cfg := config.Config{
		Communications: map[string]config.Communications{
			"comm1": {},
			"comm2": {},
		},
		Sources: map[string]config.Sources{
			"source1": {},
		},
		Executors: map[string]config.Executors{
			"exec1": {},
			"exec2": {},
			"exec3": {},
		},
	}

	reporter.UpdateConfigurationMetrics(cfg)

	commCount, err := testutil.GatherAndCount(registry, metricPrefix+"configured_communications")
	require.NoError(t, err)
	assert.Equal(t, 2, commCount)

	sourceCount, err := testutil.GatherAndCount(registry, metricPrefix+"configured_sources")
	require.NoError(t, err)
	assert.Equal(t, 1, sourceCount)

	execCount, err := testutil.GatherAndCount(registry, metricPrefix+"configured_executors")
	require.NoError(t, err)
	assert.Equal(t, 3, execCount)
	
	// Test update with fewer items
	cfg2 := config.Config{
		Communications: map[string]config.Communications{
			"comm1": {},
		},
		Sources: map[string]config.Sources{},
		Executors: map[string]config.Executors{
			"exec1": {},
		},
	}
	
	reporter.UpdateConfigurationMetrics(cfg2)
	
	commCount, err = testutil.GatherAndCount(registry, metricPrefix+"configured_communications")
	require.NoError(t, err)
	assert.Equal(t, 1, commCount)

	sourceCount, err = testutil.GatherAndCount(registry, metricPrefix+"configured_sources")
	require.NoError(t, err)
	assert.Equal(t, 0, sourceCount)

	execCount, err = testutil.GatherAndCount(registry, metricPrefix+"configured_executors")
	require.NoError(t, err)
	assert.Equal(t, 1, execCount)
}

func TestPrometheusReporter_ReportEventEvaluated(t *testing.T) {
	registry := prometheus.NewRegistry()
	
	reporter := &prometheusReporter{
		totalEvaluatedEvents: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricPrefix + "total_evaluated_events_count",
			Help: "Total number of evaluated events",
		}),
		sourceEvaluatedEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: metricPrefix + "source_evaluated_events_count",
			Help: "Number of evaluated events by source",
		}, []string{"source"}),
	}
	
	registry.MustRegister(reporter.totalEvaluatedEvents)
	registry.MustRegister(reporter.sourceEvaluatedEvents)

	reporter.ReportEventEvaluated("source1")
	reporter.ReportEventEvaluated("source2")
	reporter.ReportEventEvaluated("source1")

	totalCount := testutil.ToFloat64(reporter.totalEvaluatedEvents)
	assert.Equal(t, float64(3), totalCount)

	source1Count := testutil.ToFloat64(reporter.sourceEvaluatedEvents.WithLabelValues("source1"))
	assert.Equal(t, float64(2), source1Count)

	source2Count := testutil.ToFloat64(reporter.sourceEvaluatedEvents.WithLabelValues("source2"))
	assert.Equal(t, float64(1), source2Count)
}

func TestPrometheusReporter_ReportCommandExecuted(t *testing.T) {
	registry := prometheus.NewRegistry()
	
	reporter := &prometheusReporter{
		totalExecutedCommands: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricPrefix + "total_executed_commands_count",
			Help: "Total number of executed commands",
		}),
		executorExecutedCommands: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: metricPrefix + "executor_executed_commands_count",
			Help: "Number of executed commands by executor",
		}, []string{"executor"}),
	}
	
	registry.MustRegister(reporter.totalExecutedCommands)
	registry.MustRegister(reporter.executorExecutedCommands)

	reporter.ReportCommandExecuted("exec1")
	reporter.ReportCommandExecuted("exec2")
	reporter.ReportCommandExecuted("exec2")

	totalCount := testutil.ToFloat64(reporter.totalExecutedCommands)
	assert.Equal(t, float64(3), totalCount)

	exec1Count := testutil.ToFloat64(reporter.executorExecutedCommands.WithLabelValues("exec1"))
	assert.Equal(t, float64(1), exec1Count)

	exec2Count := testutil.ToFloat64(reporter.executorExecutedCommands.WithLabelValues("exec2"))
	assert.Equal(t, float64(2), exec2Count)
}
