package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	helpResourcesNames = []string{""}
)

// HelpExecutor executes all commands that are related to help
type HelpExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
}

// NewHelpExecutor returns a new HelpExecutor instance
func NewHelpExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter) *HelpExecutor {
	return &HelpExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
	}
}

// ResourceNames returns slice of resources the executor supports
func (e *HelpExecutor) ResourceNames() []string {
	return helpResourcesNames
}

// Commands returns slice of commands the executor supports
func (e *HelpExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandHelp: e.Help,
	}
}

// Help returns new help message
func (e *HelpExecutor) Help(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	e.reportCommand(cmdCtx.Args[0], cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	return interactive.NewHelpMessage(cmdCtx.Platform, cmdCtx.ClusterName, cmdCtx.BotName).Build(), nil
}

func (e *HelpExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting edit command: %s", err.Error())
	}
}
