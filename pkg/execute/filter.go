package execute

import (
	"bytes"
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/filterengine"
)

const (
	filterNameMissing = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled     = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled    = "Done. I won't run '%s' filter on '%s' cluster."
)

var (
	filterResourcesNames = []string{"filter", "filters", "flr"}
)

// TODO: Refactor as a part of https://github.com/kubeshop/botkube/issues/657

// FilterExecutor executes all commands that are related to filters.
type FilterExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        ConfigPersistenceManager
	filterEngine      filterengine.FilterEngine
}

// NewFilterExecutor returns a new FilterExecutor instance.
func NewFilterExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfgManager ConfigPersistenceManager, filterEngine filterengine.FilterEngine) *FilterExecutor {
	return &FilterExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfgManager:        cfgManager,
		filterEngine:      filterEngine,
	}
}

// ResourceNames returns slice of resources the executor supports
func (e *FilterExecutor) ResourceNames() []string {
	return filterResourcesNames
}

// Commands returns slice of commands the executor supports
func (e *FilterExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandList:    e.List,
		CommandEnable:  e.Enable,
		CommandDisable: e.Disable,
	}
}

// List returns a tabular representation of Filters
func (e *FilterExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	e.log.Debug("List filters")
	return respond(e.TabularOutput(), cmdCtx), nil
}

// Enable enables given filter in the startup config map
func (e *FilterExecutor) Enable(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	const enabled = true
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)

	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	if len(cmdCtx.Args) < 3 {
		return respond(fmt.Sprintf(filterNameMissing, e.TabularOutput()), cmdCtx), nil
	}
	filterName := cmdCtx.Args[2]
	e.log.Debug("Enabling filter...", filterName)
	if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
		return respond(err.Error(), cmdCtx), nil
	}

	err := e.cfgManager.PersistFilterEnabled(ctx, filterName, enabled)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
	}

	return respond(fmt.Sprintf(filterEnabled, filterName, cmdCtx.ClusterName), cmdCtx), nil
}

// Disable disables given filter in the startup config map
func (e *FilterExecutor) Disable(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	const enabled = false
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)

	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	if len(cmdCtx.Args) < 3 {
		msg := fmt.Sprintf(filterNameMissing, e.TabularOutput())
		return respond(msg, cmdCtx), nil
	}
	filterName := cmdCtx.Args[2]
	e.log.Debug("Disabling filter...", filterName)
	if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
		return respond(err.Error(), cmdCtx), nil
	}

	err := e.cfgManager.PersistFilterEnabled(ctx, filterName, enabled)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
	}

	msg := fmt.Sprintf(filterDisabled, filterName, cmdCtx.ClusterName)
	return respond(msg, cmdCtx), nil
}

// TabularOutput formats filter strings in tabular form
// https://golang.org/pkg/text/tabwriter
func (e *FilterExecutor) TabularOutput() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for _, filter := range e.filterEngine.RegisteredFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", filter.Name(), filter.Enabled, filter.Describe())
	}

	w.Flush()
	return buf.String()
}

func (e *FilterExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting edit command: %s", err.Error())
	}
}
