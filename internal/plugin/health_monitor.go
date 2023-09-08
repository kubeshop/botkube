package plugin

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
)

// HealthMonitor restarts a failed plugin process and inform scheduler to start dispatching loop again with a new client that was generated.
type HealthMonitor struct {
	log                    logrus.FieldLogger
	logConfig              config.Logger
	sourceSupervisorChan   chan pluginMetadata
	executorSupervisorChan chan pluginMetadata
	schedulerChan          chan string
	executorsStore         *store[executor.Executor]
	sourcesStore           *store[source.Source]
	policy                 config.PluginRestartPolicy
	pluginRestartStats     map[string]int
	healthCheckInterval    time.Duration
}

// NewHealthMonitor returns a new HealthMonitor instance.
func NewHealthMonitor(logger logrus.FieldLogger, logCfg config.Logger, policy config.PluginRestartPolicy, schedulerChan chan string, sourceSupervisorChan, executorSupervisorChan chan pluginMetadata, executorsStore *store[executor.Executor], sourcesStore *store[source.Source], healthCheckInterval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		log:                    logger,
		logConfig:              logCfg,
		policy:                 policy,
		schedulerChan:          schedulerChan,
		sourceSupervisorChan:   sourceSupervisorChan,
		executorSupervisorChan: executorSupervisorChan,
		executorsStore:         executorsStore,
		sourcesStore:           sourcesStore,
		pluginRestartStats:     make(map[string]int),
		healthCheckInterval:    healthCheckInterval,
	}
}

// Start starts monitor processes for sources and executors.
func (m *HealthMonitor) Start(ctx context.Context) {
	go m.monitorSourcePluginHealth(ctx)
	go m.monitorExecutorPluginHealth(ctx)
}

func (m *HealthMonitor) monitorSourcePluginHealth(ctx context.Context) {
	m.log.Info("Starting source plugin supervisor...")
	for {
		select {
		case <-ctx.Done():
			return
		case plugin := <-m.sourceSupervisorChan:
			m.log.Infof("Restarting source plugin %q, attempt %d/%d...", plugin.pluginKey, m.pluginRestartStats[plugin.pluginKey]+1, m.policy.Threshold)
			if source, ok := m.sourcesStore.EnabledPlugins.Get(plugin.pluginKey); ok && source.Cleanup != nil {
				m.log.Debugf("Releasing resources of source plugin %q...", plugin.pluginKey)
				source.Cleanup()
			}

			m.sourcesStore.EnabledPlugins.Delete(plugin.pluginKey)

			if ok := m.shouldRestartPlugin(plugin.pluginKey); !ok {
				m.log.Warnf("Plugin %q has been restarted too many times. Deactivating...", plugin.pluginKey)
				continue
			}

			p, err := createGRPCClient[source.Source](ctx, m.log, m.logConfig, plugin, TypeSource, m.sourceSupervisorChan, m.healthCheckInterval)
			if err != nil {
				m.log.WithError(err).Errorf("Failed to restart plugin %q.", plugin.pluginKey)
				continue
			}

			m.sourcesStore.EnabledPlugins.Insert(plugin.pluginKey, p)
			m.schedulerChan <- plugin.pluginKey
		}
	}
}

func (m *HealthMonitor) monitorExecutorPluginHealth(ctx context.Context) {
	m.log.Info("Starting executor plugin supervisor...")
	for {
		select {
		case <-ctx.Done():
			return
		case plugin := <-m.executorSupervisorChan:
			m.log.Infof("Restarting executor plugin %q, attempt %d/%d...", plugin.pluginKey, m.pluginRestartStats[plugin.pluginKey]+1, m.policy.Threshold)

			if executor, ok := m.executorsStore.EnabledPlugins.Get(plugin.pluginKey); ok && executor.Cleanup != nil {
				m.log.Infof("Releasing executors of executor plugin %q...", plugin.pluginKey)
				executor.Cleanup()
			}

			m.executorsStore.EnabledPlugins.Delete(plugin.pluginKey)
			if ok := m.shouldRestartPlugin(plugin.pluginKey); !ok {
				m.log.Warnf("Plugin %q has been restarted too many times. Deactivating...", plugin.pluginKey)
				continue
			}

			p, err := createGRPCClient[executor.Executor](ctx, m.log, m.logConfig, plugin, TypeExecutor, m.executorSupervisorChan, m.healthCheckInterval)
			if err != nil {
				m.log.WithError(err).Errorf("Failed to restart plugin %q.", plugin.pluginKey)
				continue
			}

			m.executorsStore.EnabledPlugins.Insert(plugin.pluginKey, p)
		}
	}
}

func (m *HealthMonitor) shouldRestartPlugin(plugin string) bool {
	restarts := m.pluginRestartStats[plugin]
	m.pluginRestartStats[plugin]++

	switch m.policy.Type.ToLower() {
	case config.KeepAgentRunningWhenThresholdReached.ToLower():
		return restarts < m.policy.Threshold
	case config.RestartAgentWhenThresholdReached.ToLower():
		if restarts >= m.policy.Threshold {
			m.log.Fatalf("Plugin %q has been restarted %d times and selected restartPolicy is %q. Exiting...", plugin, restarts, m.policy.Type)
		}
		return true
	default:
		m.log.Errorf("Unknown restart policy %q.", m.policy.Type)
		return false
	}
}
