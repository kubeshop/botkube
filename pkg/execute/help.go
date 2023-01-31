package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	helpFeatureName = FeatureName{Name: noFeature}
)

// HelpExecutor executes all commands that are related to help
type HelpExecutor struct {
	log                    logrus.FieldLogger
	analyticsReporter      AnalyticsReporter
	enabledPluginExecutors []string
}

// NewHelpExecutor returns a new HelpExecutor instance
func NewHelpExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfg config.Config) *HelpExecutor {
	collector := plugin.NewCollector(log)
	enabledPluginExecutors, _ := collector.GetAllEnabledAndUsedPlugins(&cfg)

	return &HelpExecutor{
		log:                    log,
		analyticsReporter:      analyticsReporter,
		enabledPluginExecutors: enabledPluginExecutors,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *HelpExecutor) FeatureName() FeatureName {
	return helpFeatureName
}

// Commands returns slice of commands the executor supports
func (e *HelpExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.HelpVerb: e.Help,
	}
}

// Help returns new help message
func (e *HelpExecutor) Help(_ context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, _ := parseCmdVerb(cmdCtx.Args)
	e.reportCommand(cmdVerb, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	return interactive.NewHelpMessage(cmdCtx.Platform, cmdCtx.ClusterName, cmdCtx.BotName, e.enabledPluginExecutors).Build(), nil
}

func (e *HelpExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting help command: %s", err.Error())
	}
}
