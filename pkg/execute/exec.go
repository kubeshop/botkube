package execute

import (
	"bytes"
	"context"
	"fmt"
	"github.com/kubeshop/botkube/pkg/execute/alias"
	"github.com/kubeshop/botkube/pkg/maputil"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	executorFeatureName = FeatureName{
		Name:    "executor",
		Aliases: []string{"executors", "exec"},
	}
)

const kubectlBuiltinExecutorName = "kubectl"

// ExecExecutor executes all commands that are related to executors.
type ExecExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfg               config.Config
}

// NewExecExecutor returns a new ExecExecutor instance.
func NewExecExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfg config.Config) *ExecExecutor {
	return &ExecExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfg:               cfg,
	}
}

// Commands returns slice of commands the executor supports
func (e *ExecExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandList: e.List,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *ExecExecutor) FeatureName() FeatureName {
	return executorFeatureName
}

// List returns a tabular representation of Executors
func (e *ExecExecutor) List(_ context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	e.log.Debug("List executors")
	return respond(e.TabularOutput(cmdCtx.Conversation.ExecutorBindings), cmdCtx), nil
}

// TabularOutput sorts executor groups by key and returns a printable table
func (e *ExecExecutor) TabularOutput(bindings []string) string {
	executorsForBindings := make(map[string]bool)

	for _, b := range bindings {
		executor, ok := e.cfg.Executors[b]
		if !ok {
			continue
		}

		for name, plugin := range executor.Plugins {
			executorsForBindings[name] = plugin.Enabled
		}

		// TODO: Remove once kubectl is migrated to a separate plugin
		if executor.Kubectl.Enabled && !executorsForBindings[kubectlBuiltinExecutorName] {
			executorsForBindings[kubectlBuiltinExecutorName] = true
		}
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "EXECUTOR\tENABLED\tALIASES")
	for _, name := range maputil.SortKeys(executorsForBindings) {
		enabled := executorsForBindings[name]
		aliases := alias.ListForExecutor(name, e.cfg.Aliases)
		fmt.Fprintf(w, "%s\t%t\t%s\n", name, enabled, strings.Join(aliases, ", "))
	}

	w.Flush()
	return buf.String()
}

func (e *ExecExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting executor command: %s", err.Error())
	}
}
