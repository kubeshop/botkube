package execute

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/filterengine"
)

const (
	filterNameMissing = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled     = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled    = "Done. I won't run '%s' filter on '%s' cluster."
)

var (
	filterFeatureName = FeatureName{
		Name:    "filter",
		Aliases: []string{"filters", "flr"},
	}
)

// TODO: Refactor as a part of https://github.com/kubeshop/botkube/issues/657

// FilterExecutor executes all commands that are related to filters.
type FilterExecutor struct {
	log          logrus.FieldLogger
	cfgManager   ConfigPersistenceManager
	filterEngine filterengine.FilterEngine
}

// NewFilterExecutor returns a new FilterExecutor instance.
func NewFilterExecutor(log logrus.FieldLogger, cfgManager ConfigPersistenceManager, filterEngine filterengine.FilterEngine) *FilterExecutor {
	return &FilterExecutor{
		log:          log,
		cfgManager:   cfgManager,
		filterEngine: filterEngine,
	}
}

// Commands returns slice of commands the executor supports
func (e *FilterExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.ListVerb:    e.List,
		command.EnableVerb:  e.Enable,
		command.DisableVerb: e.Disable,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *FilterExecutor) FeatureName() FeatureName {
	return filterFeatureName
}

// List returns a tabular representation of Filters
func (e *FilterExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	e.log.Debug("List filters")
	return respond(e.TabularOutput(), cmdCtx), nil
}

// Enable enables given filter in the startup config map
func (e *FilterExecutor) Enable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = true
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
		return interactive.CoreMessage{}, fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
	}

	return respond(fmt.Sprintf(filterEnabled, filterName, cmdCtx.ClusterName), cmdCtx), nil
}

// Disable disables given filter in the startup config map
func (e *FilterExecutor) Disable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = false
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
		return interactive.CoreMessage{}, fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
	}

	msg := fmt.Sprintf(filterDisabled, filterName, cmdCtx.ClusterName)
	return respond(msg, cmdCtx), nil
}

// TabularOutput formats filter strings in tabular form
// https://golang.org/pkg/text/tabwriter
func (e *FilterExecutor) TabularOutput() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintf(w, "FILTER\tENABLED\tDESCRIPTION")
	for _, filter := range e.filterEngine.RegisteredFilters() {
		fmt.Fprintf(w, "\n%s\t%v\t%s", filter.Name(), filter.Enabled, filter.Describe())
	}

	w.Flush()
	return buf.String()
}

func appendInteractiveFilterIfNeeded(body string, msg interactive.CoreMessage, cmdCtx CommandContext) interactive.CoreMessage {
	if !cmdCtx.Platform.IsInteractive() {
		return msg
	}
	if len(strings.SplitN(body, "\n", lineLimitToShowFilter)) < lineLimitToShowFilter {
		return msg
	}

	msg.PlaintextInputs = append(msg.PlaintextInputs, filterInput(cmdCtx.CleanCmd))
	return msg
}
