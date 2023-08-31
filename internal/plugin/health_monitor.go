package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
)

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
			m.log.Infof("Restarting source plugin %q, attempt %d/%d...", plugin.name, m.pluginRestartStats[plugin.name]+1, m.policy.Threshold)
			if source, ok := m.sourcesStore.EnabledPlugins.Get(plugin.name); ok && source.Cleanup != nil {
				m.log.Debugf("Releasing resources of source plugin %q...", plugin.name)
				source.Cleanup()
			}

			// botkube/kubernetes
			repoPluginPair := fmt.Sprintf("%s/%s", plugin.repo, plugin.name)
			m.sourcesStore.EnabledPlugins.Delete(repoPluginPair)

			if ok := m.shouldRestartPlugin(repoPluginPair); !ok {
				m.log.Warnf("Plugin %q has been restarted too many times. Deactivating...", plugin.name)
				continue
			}

			p, err := createGRPCClient[source.Source](ctx, m.log, m.logConfig, plugin, TypeSource, m.sourceSupervisorChan, m.healthCheckInterval)
			if err != nil {
				m.log.WithError(err).Errorf("Failed to restart plugin %q.", plugin.name)
				continue
			}

			m.sourcesStore.EnabledPlugins.Insert(repoPluginPair, p)
			m.schedulerChan <- fmt.Sprintf("%s/%s/%s", plugin.group, plugin.repo, plugin.name)
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
			m.log.Infof("Restarting executor plugin %q, attempt %d/%d...", plugin.name, m.pluginRestartStats[plugin.name]+1, m.policy.Threshold)
			if executor, ok := m.executorsStore.EnabledPlugins.Get(plugin.name); ok && executor.Cleanup != nil {
				m.log.Infof("Releasing executors of executor plugin %q...", plugin.name)
				executor.Cleanup()
			}

			// botkube/kubectl
			// note: if other naming scheme is used, it might be safer to try guess the name from channel bindings
			repoPluginPair := fmt.Sprintf("%s/%s", plugin.repo, plugin.name)
			m.executorsStore.EnabledPlugins.Delete(repoPluginPair)

			if ok := m.shouldRestartPlugin(repoPluginPair); !ok {
				m.log.Warnf("Plugin %q has been restarted too many times. Deactivating...", plugin.name)
				continue
			}

			p, err := createGRPCClient[executor.Executor](ctx, m.log, m.logConfig, plugin, TypeExecutor, m.executorSupervisorChan, m.healthCheckInterval)
			if err != nil {
				m.log.WithError(err).Errorf("Failed to restart plugin %q.", plugin.name)
				continue
			}

			m.executorsStore.EnabledPlugins.Insert(repoPluginPair, p)
		}
	}
}

func (m *HealthMonitor) shouldRestartPlugin(plugin string) bool {
	restarts := m.pluginRestartStats[plugin]
	m.pluginRestartStats[plugin]++

	switch m.policy.Type {
	case config.KeepAgentRunningWhenThresholdReached:
		return restarts < m.policy.Threshold
	case config.RestartAgentWhenThresholdReached:
		if restarts >= m.policy.Threshold {
			m.log.Fatalf("Plugin %q has been restarted %d times and selected restartPolicy is %q. Exiting...", plugin, restarts, m.policy.Type)
		}
		return true
	default:
		m.log.Errorf("Unknown restart policy %q.", m.policy.Type)
		return false
	}
}
