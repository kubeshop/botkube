package execute

import "github.com/kubeshop/botkube/pkg/config"

type fakeAnalyticsReporter struct{}

func (f *fakeAnalyticsReporter) ReportCommand(_ config.CommPlatformIntegration, _ string) error {
	return nil
}

type fakeCfgPersistenceManager struct{}

func (f *fakeCfgPersistenceManager) PersistSourceBindings(commGroupName string, platform config.CommPlatformIntegration, channelName string, sourceBindings []string) error {
	return nil
}

func (f *fakeCfgPersistenceManager) PersistNotificationsEnabled(commGroupName string, platform config.CommPlatformIntegration, channelName string, enabled bool) error {
	return nil
}

func (f *fakeCfgPersistenceManager) PersistFilterEnabled(name string, enabled bool) error {
	return nil
}
