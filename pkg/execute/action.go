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
	"github.com/kubeshop/botkube/pkg/maputil"
)

const (
	actionNameMissing = "You forgot to pass action name. Please pass one of the following valid actions:\n\n%s"
	actionEnabled     = "I have enabled '%s' action on '%s' cluster."
	actionDisabled    = "Done. I won't run '%s' action on '%s' cluster."
)

var (
	actionFeatureName = FeatureName{
		Name:    "action",
		Aliases: []string{"actions", "act"},
	}
)

type ActionsStorage interface {
	PersistActionEnabled(ctx context.Context, name string, enabled bool) error
}

// ActionExecutor executes all commands that are related to actions.
type ActionExecutor struct {
	log        logrus.FieldLogger
	cfgManager ActionsStorage
	actions    map[string]config.Action
}

// NewActionExecutor returns a new ActionExecutor instance.
func NewActionExecutor(log logrus.FieldLogger, cfgManager ActionsStorage, cfg config.Config) *ActionExecutor {
	return &ActionExecutor{
		log:        log,
		cfgManager: cfgManager,
		actions:    cfg.Actions,
	}
}

// Commands returns slice of commands the executor supports
func (e *ActionExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.ListVerb:    e.List,
		command.EnableVerb:  e.Enable,
		command.DisableVerb: e.Disable,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *ActionExecutor) FeatureName() FeatureName {
	return actionFeatureName
}

// List returns a tabular representation of Actions
func (e *ActionExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	e.log.Debug("List actions")
	return respond(e.ActionsTabularOutput(), cmdCtx), nil
}

// Enable enables given action in the runtime config map
func (e *ActionExecutor) Enable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = true
	if len(cmdCtx.Args) < 3 {
		return respond(fmt.Sprintf(actionNameMissing, e.ActionsTabularOutput()), cmdCtx), nil
	}
	actionName := cmdCtx.Args[2]
	e.log.Debug("Enabling action...", actionName)

	if err := e.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while setting action %q to %t: %w", actionName, enabled, err)
	}
	return respond(fmt.Sprintf(actionEnabled, actionName, cmdCtx.ClusterName), cmdCtx), nil
}

// Disable disables given action in the runtime config map
func (e *ActionExecutor) Disable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = false
	if len(cmdCtx.Args) < 3 {
		return respond(fmt.Sprintf(actionNameMissing, e.ActionsTabularOutput()), cmdCtx), nil
	}
	actionName := cmdCtx.Args[2]
	e.log.Debug("Disabling action...", actionName)

	if err := e.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while setting action %q to %t: %w", actionName, enabled, err)
	}
	return respond(fmt.Sprintf(actionDisabled, actionName, cmdCtx.ClusterName), cmdCtx), nil
}

// ActionsTabularOutput sorts actions by key and returns a printable table
func (e *ActionExecutor) ActionsTabularOutput() string {
	keys := maputil.SortKeys(e.actions)

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintf(w, "ACTION\tENABLED \tDISPLAY NAME")
	for _, name := range keys {
		fmt.Fprintf(w, "\n%s\t%v \t%s", name, e.actions[name].Enabled, e.actions[name].DisplayName)
	}
	w.Flush()
	return buf.String()
}
