package execute

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/audit"
	guard "github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

// DefaultExecutorFactory facilitates creation of the Executor instances.
type DefaultExecutorFactory struct {
	log                   logrus.FieldLogger
	cfg                   config.Config
	analyticsReporter     AnalyticsReporter
	notifierExecutor      *NotifierExecutor
	pluginExecutor        *PluginExecutor
	sourceBindingExecutor *SourceBindingExecutor
	actionExecutor        *ActionExecutor
	pingExecutor          *PingExecutor
	versionExecutor       *VersionExecutor
	helpExecutor          *HelpExecutor
	feedbackExecutor      *FeedbackExecutor
	configExecutor        *ConfigExecutor
	execExecutor          *ExecExecutor
	sourceExecutor        *SourceExecutor
	cmdsMapping           *CommandMapping
	auditReporter         audit.AuditReporter
	pluginHealthStats     *plugin.HealthStats
}

// DefaultExecutorFactoryParams contains input parameters for DefaultExecutorFactory.
type DefaultExecutorFactoryParams struct {
	Log               logrus.FieldLogger
	Cfg               config.Config
	CfgManager        config.PersistenceManager
	AnalyticsReporter AnalyticsReporter
	CommandGuard      CommandGuard
	PluginManager     *plugin.Manager
	RestCfg           *rest.Config
	BotKubeVersion    string
	AuditReporter     audit.AuditReporter
	PluginHealthStats *plugin.HealthStats
}

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute(context.Context) interactive.CoreMessage
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportCommand reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(platform config.CommPlatformIntegration, command string, origin command.Origin, withFilter bool) error
}

// CommandGuard is an interface that allows to check if a given command is allowed to be executed.
type CommandGuard interface {
	GetAllowedResourcesForVerb(verb string, allConfiguredResources []string) ([]guard.Resource, error)
	GetResourceDetails(verb, resourceType string) (guard.Resource, error)
	FilterSupportedVerbs(allVerbs []string) []string
}

// NewExecutorFactory creates new DefaultExecutorFactory.
func NewExecutorFactory(params DefaultExecutorFactoryParams) (*DefaultExecutorFactory, error) {
	actionExecutor := NewActionExecutor(
		params.Log.WithField("component", "Action Executor"),
		params.CfgManager,
		params.Cfg,
	)
	sourceBindingExecutor := NewSourceBindingExecutor(
		params.Log.WithField("component", "SourceBinding Executor"),
		params.CfgManager,
		params.Cfg,
	)
	pingExecutor := NewPingExecutor(
		params.Log.WithField("component", "Ping Executor"),
		params.BotKubeVersion,
	)
	versionExecutor := NewVersionExecutor(
		params.Log.WithField("component", "Version Executor"),
		params.BotKubeVersion,
	)
	feedbackExecutor := NewFeedbackExecutor(
		params.Log.WithField("component", "Feedback Executor"),
	)
	notifierExecutor := NewNotifierExecutor(
		params.Log.WithField("component", "Notifier Executor"),
		params.CfgManager,
	)
	helpExecutor := NewHelpExecutor(
		params.Log.WithField("component", "Help Executor"),
		params.Cfg,
	)
	configExecutor := NewConfigExecutor(
		params.Log.WithField("component", "Config Executor"),
		params.Cfg,
	)
	execExecutor := NewExecExecutor(
		params.Log.WithField("component", "Executor Bindings Executor"),
		params.Cfg,
	)
	sourceExecutor := NewSourceExecutor(
		params.Log.WithField("component", "Source Bindings Executor"),
		params.Cfg,
	)
	aliasExecutor := NewAliasExecutor(
		params.Log.WithField("component", "Alias Executor"),
		params.Cfg,
	)

	executors := []CommandExecutor{
		actionExecutor,
		sourceBindingExecutor,
		pingExecutor,
		versionExecutor,
		helpExecutor,
		feedbackExecutor,
		notifierExecutor,
		configExecutor,
		execExecutor,
		sourceExecutor,
		aliasExecutor,
	}
	mappings, err := NewCmdsMapping(executors)
	if err != nil {
		return nil, err
	}
	return &DefaultExecutorFactory{
		log:               params.Log,
		cfg:               params.Cfg,
		analyticsReporter: params.AnalyticsReporter,
		notifierExecutor:  notifierExecutor,
		pluginExecutor: NewPluginExecutor(
			params.Log.WithField("component", "Botkube Plugin Executor"),
			params.Cfg,
			params.PluginManager,
			params.RestCfg,
		),
		sourceBindingExecutor: sourceBindingExecutor,
		actionExecutor:        actionExecutor,
		pingExecutor:          pingExecutor,
		versionExecutor:       versionExecutor,
		helpExecutor:          helpExecutor,
		feedbackExecutor:      feedbackExecutor,
		configExecutor:        configExecutor,
		execExecutor:          execExecutor,
		sourceExecutor:        sourceExecutor,
		cmdsMapping:           mappings,
		auditReporter:         params.AuditReporter,
		pluginHealthStats:     params.PluginHealthStats,
	}, nil
}

// Conversation contains details about the conversation.
type Conversation struct {
	Alias            string
	DisplayName      string
	ID               string
	ExecutorBindings []string
	SourceBindings   []string
	IsKnown          bool
	CommandOrigin    command.Origin
	SlackState       *slack.BlockActionStates
}

// NewDefaultInput an input for NewDefault
type NewDefaultInput struct {
	CommGroupName   string
	Platform        config.CommPlatformIntegration
	NotifierHandler NotifierHandler
	Conversation    Conversation
	Message         string
	User            UserInput
}

// UserInput contains details about the user.
type UserInput struct {
	Mention     string
	DisplayName string
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(cfg NewDefaultInput) Executor {
	return &DefaultExecutor{
		log:                   f.log,
		cfg:                   f.cfg,
		analyticsReporter:     f.analyticsReporter,
		pluginExecutor:        f.pluginExecutor,
		notifierExecutor:      f.notifierExecutor,
		sourceBindingExecutor: f.sourceBindingExecutor,
		actionExecutor:        f.actionExecutor,
		pingExecutor:          f.pingExecutor,
		versionExecutor:       f.versionExecutor,
		helpExecutor:          f.helpExecutor,
		feedbackExecutor:      f.feedbackExecutor,
		configExecutor:        f.configExecutor,
		execExecutor:          f.execExecutor,
		sourceExecutor:        f.sourceExecutor,
		cmdsMapping:           f.cmdsMapping,
		auditReporter:         f.auditReporter,
		pluginHealthStats:     f.pluginHealthStats,
		user:                  cfg.User,
		notifierHandler:       cfg.NotifierHandler,
		conversation:          cfg.Conversation,
		message:               cfg.Message,
		platform:              cfg.Platform,
		commGroupName:         cfg.CommGroupName,
	}
}
