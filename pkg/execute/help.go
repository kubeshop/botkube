package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/plugin"
)

var (
	helpFeatureName = FeatureName{Name: noFeature}
)

// HelpExecutor executes all commands that are related to help
type HelpExecutor struct {
	log                    logrus.FieldLogger
	enabledPluginExecutors []string
}

// NewHelpExecutor returns a new HelpExecutor instance
func NewHelpExecutor(log logrus.FieldLogger, cfg config.Config) *HelpExecutor {
	collector := plugin.NewCollector(log)
	enabledPluginExecutors, _ := collector.GetAllEnabledAndUsedPlugins(&cfg)

	return &HelpExecutor{
		log:                    log,
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
func (e *HelpExecutor) Help(_ context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	return interactive.NewHelpMessage(cmdCtx.Platform, cmdCtx.ClusterName, e.enabledPluginExecutors).Build(false), nil
}
