package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
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
	editExecutor      *EditExecutor
	merger            *kubectl.Merger
	cfgManager        ConfigPersistenceManager
}

// DefaultExecutorFactoryParams contains input parameters for DefaultExecutorFactory.
type DefaultExecutorFactoryParams struct {
	Log               logrus.FieldLogger
	CmdRunner         CommandRunner
	Cfg               config.Config
	FilterEngine      filterengine.FilterEngine
	KcChecker         *kubectl.Checker
	Merger            *kubectl.Merger
	CfgManager        ConfigPersistenceManager
	AnalyticsReporter AnalyticsReporter
}

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() interactive.Message
}

// ConfigPersistenceManager manages persistence of the configuration.
type ConfigPersistenceManager interface {
	PersistSourceBindings(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, sourceBindings []string) error
	PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, enabled bool) error
	PersistFilterEnabled(ctx context.Context, name string, enabled bool) error
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportCommand reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(platform config.CommPlatformIntegration, command string) error
}

// NewExecutorFactory creates new DefaultExecutorFactory.
func NewExecutorFactory(params DefaultExecutorFactoryParams) *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		log:               params.Log,
		cmdRunner:         params.CmdRunner,
		cfg:               params.Cfg,
		filterEngine:      params.FilterEngine,
		analyticsReporter: params.AnalyticsReporter,
		notifierExecutor: NewNotifierExecutor(
			params.Log.WithField("component", "Notifier Executor"),
			params.Cfg,
			params.CfgManager,
			params.AnalyticsReporter,
		),
		editExecutor: NewEditExecutor(
			params.Log.WithField("component", "Notifier Executor"),
			params.AnalyticsReporter,
			params.CfgManager,
			params.Cfg,
		),
		merger:     params.Merger,
		cfgManager: params.CfgManager,
		kubectlExecutor: NewKubectl(
			params.Log.WithField("component", "Kubectl Executor"),
			params.Cfg,
			params.Merger,
			params.KcChecker,
			params.CmdRunner,
		),
	}
}

// Conversation contains details about the conversation.
type Conversation struct {
	Alias            string
	ID               string
	ExecutorBindings []string
	IsAuthenticated  bool
}

// NewDefaultInput an input for NewDefault
type NewDefaultInput struct {
	CommGroupName   string
	Platform        config.CommPlatformIntegration
	NotifierHandler NotifierHandler
	Conversation    Conversation
	Message         string
	User            string
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(cfg NewDefaultInput) Executor {
	return &DefaultExecutor{
		log:               f.log,
		cmdRunner:         f.cmdRunner,
		cfg:               f.cfg,
		analyticsReporter: f.analyticsReporter,
		kubectlExecutor:   f.kubectlExecutor,
		notifierExecutor:  f.notifierExecutor,
		editExecutor:      f.editExecutor,
		filterEngine:      f.filterEngine,
		merger:            f.merger,
		cfgManager:        f.cfgManager,
		user:              cfg.User,
		notifierHandler:   cfg.NotifierHandler,
		conversation:      cfg.Conversation,
		message:           cfg.Message,
		platform:          cfg.Platform,
		commGroupName:     cfg.CommGroupName,
	}
}
