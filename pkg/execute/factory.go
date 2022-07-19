package execute

import (
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/filterengine"
)

// DefaultExecutorFactory facilitates creation of the Executor instances.
type DefaultExecutorFactory struct {
	log               logrus.FieldLogger
	runCmdFn          CommandRunnerFunc
	cfg               config.Config
	filterEngine      filterengine.FilterEngine
	resMapping        ResourceMapping
	analyticsReporter AnalyticsReporter
}

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportCommand reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(platform config.CommPlatformIntegration, command string) error
}

// NewExecutorFactory creates new DefaultExecutorFactory.
func NewExecutorFactory(
	log logrus.FieldLogger,
	runCmdFn CommandRunnerFunc,
	cfg config.Config,
	filterEngine filterengine.FilterEngine,
	resMapping ResourceMapping,
	analyticsReporter AnalyticsReporter,
) *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		log:               log,
		runCmdFn:          runCmdFn,
		cfg:               cfg,
		filterEngine:      filterEngine,
		resMapping:        resMapping,
		analyticsReporter: analyticsReporter,
	}
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(platform config.CommPlatformIntegration, isAuthChannel bool, message string) Executor {
	return &DefaultExecutor{
		log:               f.log,
		runCmdFn:          f.runCmdFn,
		cfg:               f.cfg,
		resMapping:        f.resMapping,
		analyticsReporter: f.analyticsReporter,

		filterEngine:  f.filterEngine,
		IsAuthChannel: isAuthChannel,
		Message:       message,
		Platform:      platform,
	}
}
