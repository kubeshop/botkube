package execute

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const (
	// Currently we support only `kubectl, so we
	// override the message to human-readable command name.
	humanReadableCommandListName = "Available kubectl commands"
)

var (
	commandFeatureName = FeatureName{
		Name:    "command",
		Aliases: []string{"commands", "cmd", "cmds"},
	}
)

// CommandsExecutor executes all commands that are related to command
type CommandsExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	merger            *kubectl.Merger
}

// NewCommandsExecutor returns a new CommandsExecutor instance
func NewCommandsExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, merger *kubectl.Merger) *CommandsExecutor {
	return &CommandsExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		merger:            merger,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *CommandsExecutor) FeatureName() FeatureName {
	return commandFeatureName
}

// Commands returns slice of commands the executor supports
func (e *CommandsExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandList: e.List,
	}
}

// List provides the list of allowed commands
func (e *CommandsExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	enabledKubectls, err := e.getEnabledKubectlExecutorsInChannel(cmdCtx.Conversation.ExecutorBindings)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while rendering enabled executors: %s", err.Error())
	}
	return respond(enabledKubectls, cmdCtx, humanReadableCommandListName), nil
}

func (e *CommandsExecutor) getEnabledKubectlExecutorsInChannel(executorBindings []string) (string, error) {
	type kubectlCollection map[string]config.Kubectl
	enabledKubectls := e.merger.GetAllEnabled(executorBindings)
	out := map[string]map[string]kubectlCollection{
		"Enabled executors": {
			"kubectl": enabledKubectls,
		},
	}

	var buff strings.Builder
	encode := yaml.NewEncoder(&buff)
	encode.SetIndent(2)
	err := encode.Encode(out)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}

func (e *CommandsExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting %s command: %s", cmdToReport, err.Error())
	}
}
