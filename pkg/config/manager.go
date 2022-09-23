package config

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// PersistenceManager manages persistence of the configuration.
type PersistenceManager struct {
	log    logrus.FieldLogger
	k8sCli kubernetes.Interface
}

// NewManager creates a new PersistenceManager instance.
func NewManager(log logrus.FieldLogger, k8sCli kubernetes.Interface) *PersistenceManager {
	return &PersistenceManager{log: log, k8sCli: k8sCli}
}

// PersistSourceBindings persists source bindings configuration for a given channel in a given platform.
func (m *PersistenceManager) PersistSourceBindings(commGroupName string, platform CommPlatformIntegration, channelName string, sourceBindings []string) error {
	// TODO: Implement this as a part of https://github.com/kubeshop/botkube/issues/704
	m.log.WithFields(
		logrus.Fields{
			"commGroupName":  commGroupName,
			"platform":       platform,
			"channelName":    channelName,
			"sourceBindings": sourceBindings,
		},
	).Info("PersistSourceBindings called")

	return nil
}

// PersistNotificationsEnabled persists notifications state for a given channel.
// While this method updates the BotKube ConfigMap, it doesn't reload BotKube itself.
func (m *PersistenceManager) PersistNotificationsEnabled(commGroupName string, platform CommPlatformIntegration, channelName string, enabled bool) error {
	// TODO: Implement this as a part of https://github.com/kubeshop/botkube/issues/704
	return nil
}

// PersistFilterEnabled persists status for a given filter.
// While this method updates the BotKube ConfigMap, it doesn't reload BotKube itself.
func (m *PersistenceManager) PersistFilterEnabled(name string, enabled bool) error {
	// TODO: Implement this as a part of https://github.com/kubeshop/botkube/issues/704
	return nil
}
