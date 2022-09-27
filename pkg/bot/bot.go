package bot

import (
	"context"
	"github.com/kubeshop/botkube/pkg/notifier"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
)

// Bot connects to communication channels and reads/sends messages. It is a two-way integration.
type Bot interface {
	Start(ctx context.Context) error
	BotName() string
	notifier.Notifier
}

// ExecutorFactory facilitates creation of execute.Executor instances.
type ExecutorFactory interface {
	NewDefault(cfg execute.NewDefaultInput) execute.Executor
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportBotEnabled reports an enabled bot.
	ReportBotEnabled(platform config.CommPlatformIntegration) error
}

// FatalErrorAnalyticsReporter reports a fatal errors.
type FatalErrorAnalyticsReporter interface {
	AnalyticsReporter

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

type channelConfigByID struct {
	config.ChannelBindingsByID

	alias  string
	notify bool
}

type channelConfigByName struct {
	config.ChannelBindingsByName

	alias  string
	notify bool
}
