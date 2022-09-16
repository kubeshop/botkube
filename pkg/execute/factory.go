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
	cmdRunner         CommandSeparateOutputRunner
	cfg               config.Config
	filterEngine      filterengine.FilterEngine
	analyticsReporter AnalyticsReporter
	notifierExecutor  *NotifierExecutor
	kubectlExecutor   *Kubectl
	merger            *kubectl.Merger
	cfgManager        ConfigPersistenceManager
}

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// ConfigPersistenceManager manages persistence of the configuration.
type ConfigPersistenceManager interface {
	PersistSourceBindings(commGroupName string, platform config.CommPlatformIntegration, channelName string, sourceBindings []string) error
	PersistNotificationsEnabled(commGroupName string, platform config.CommPlatformIntegration, channelName string, enabled bool) error
	PersistFilterEnabled(name string, enabled bool) error
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportCommand reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(platform config.CommPlatformIntegration, command string) error
}

// NewExecutorFactory creates new DefaultExecutorFactory.
func NewExecutorFactory(log logrus.FieldLogger, cmdRunner CommandRunner, cfg config.Config, filterEngine filterengine.FilterEngine, kcChecker *kubectl.Checker, merger *kubectl.Merger, cfgManager ConfigPersistenceManager, analyticsReporter AnalyticsReporter) *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		log:               log,
		cmdRunner:         cmdRunner,
		cfg:               cfg,
		filterEngine:      filterEngine,
		analyticsReporter: analyticsReporter,
		notifierExecutor: NewNotifierExecutor(
			log.WithField("component", "Notifier Executor"),
			cfg,
			cfgManager,
			analyticsReporter,
		),
		merger:     merger,
		cfgManager: cfgManager,
		kubectlExecutor: NewKubectl(
			log.WithField("component", "Kubectl Executor"),
			cfg,
			merger,
			kcChecker,
			cmdRunner,
		),
	}
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(commGroupName string, platform config.CommPlatformIntegration, notifierHandler NotifierHandler, isAuthChannel bool, conversationID string, bindings []string, message string) Executor {
	return &DefaultExecutor{
		log:               f.log,
		cmdRunner:         f.cmdRunner,
		cfg:               f.cfg,
		analyticsReporter: f.analyticsReporter,
		kubectlExecutor:   f.kubectlExecutor,
		notifierExecutor:  f.notifierExecutor,
		filterEngine:      f.filterEngine,
		merger:            f.merger,
		cfgManager:        f.cfgManager,
		notifierHandler:   notifierHandler,
		isAuthChannel:     isAuthChannel,
		bindings:          bindings,
		message:           message,
		platform:          platform,
		conversationID:    conversationID,
		commGroupName:     commGroupName,
	}
}
