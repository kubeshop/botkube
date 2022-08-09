package execute

import (
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/filterengine"
)

// DefaultExecutorFactory facilitates creation of the Executor instances.
type DefaultExecutorFactory struct {
	log               logrus.FieldLogger
	runCmdFn          CommandRunnerFunc
	cfg               config.Config
	filterEngine      filterengine.FilterEngine
	analyticsReporter AnalyticsReporter
	notifierExecutor  *NotifierExecutor
	kubectlExecutor   *Kubectl
	merger            *kubectl.Merger
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
func NewExecutorFactory(log logrus.FieldLogger, runCmdFn CommandRunnerFunc, cfg config.Config, filterEngine filterengine.FilterEngine, kcChecker *kubectl.Checker, merger *kubectl.Merger, analyticsReporter AnalyticsReporter) *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		log:               log,
		runCmdFn:          runCmdFn,
		cfg:               cfg,
		filterEngine:      filterEngine,
		analyticsReporter: analyticsReporter,
		notifierExecutor: NewNotifierExecutor(
			log.WithField("component", "Notifier Executor"),
			cfg,
			analyticsReporter,
		),
		merger: merger,
		kubectlExecutor: NewKubectl(
			log.WithField("component", "Kubectl Executor"),
			cfg,
			merger,
			kcChecker,
			runCmdFn,
		),
	}
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(platform config.CommPlatformIntegration, notifierHandler NotifierHandler, isAuthChannel bool, conversationID string, bindings []string, message string) Executor {
	return &DefaultExecutor{
		log:               f.log,
		runCmdFn:          f.runCmdFn,
		cfg:               f.cfg,
		analyticsReporter: f.analyticsReporter,
		kubectlExecutor:   f.kubectlExecutor,
		notifierExecutor:  f.notifierExecutor,
		filterEngine:      f.filterEngine,
		merger:            f.merger,
		notifierHandler:   notifierHandler,
		isAuthChannel:     isAuthChannel,
		bindings:          bindings,
		message:           message,
		platform:          platform,
		conversationID:    conversationID,
	}
}
