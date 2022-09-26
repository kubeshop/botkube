package execute

import (
	"context"
	"errors"

	"github.com/kubeshop/botkube/pkg/config"
)

type fakeAnalyticsReporter struct{}

func (f *fakeAnalyticsReporter) ReportCommand(_ config.CommPlatformIntegration, _ string) error {
	return nil
}

type fakeCfgPersistenceManager struct {
	expectedAlias string
}

func (f *fakeCfgPersistenceManager) PersistSourceBindings(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, sourceBindings []string) error {
	if f.expectedAlias != channelAlias {
		return errors.New("different alias")
	}
	return nil
}

func (f *fakeCfgPersistenceManager) PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, enabled bool) error {
	if f.expectedAlias != channelAlias {
		return errors.New("different alias")
	}
	return nil
}

func (f *fakeCfgPersistenceManager) PersistFilterEnabled(ctx context.Context, name string, enabled bool) error {
	return nil
}
