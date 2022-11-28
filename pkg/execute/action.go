package execute

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

var actionResourceNames = map[string]struct{}{
	"action":  {},
	"act":     {},
	"actions": {},
}

// ActionExecutor executes all commands that are related to actions.
type ActionExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        ConfigPersistenceManager
	actions           map[string]config.Action
}

// NewActionExecutor returns a new ActionExecutor instance.
func NewActionExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfgManager ConfigPersistenceManager, cfg config.Config) *ActionExecutor {
	return &ActionExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfgManager:        cfgManager,
		actions:           cfg.Actions,
	}
}

// Do executes a given action command based on args
func (a *ActionExecutor) Do(ctx context.Context, args []string, clusterName string, conversation Conversation, platform config.CommPlatformIntegration) (string, error) {
	if len(args) < 2 {
		return "", errInvalidCommand
	}

	cmdRes := strings.ToLower(args[1])
	_, ok := actionResourceNames[cmdRes]
	if !ok {
		return fmt.Sprintf("'%s' is not a valid resource", args[1]), nil
	}

	var cmdVerb = args[0]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
		e := a.analyticsReporter.ReportCommand(platform, cmdToReport, conversation.CommandOrigin, false)
		if e != nil {
			a.log.Errorf("while reporting edit command: %s", e.Error())
		}
	}()

	switch CommandVerb(cmdVerb) {
	case CommandList:
		a.log.Debug("List actions")
		return a.ActionsTabularOutput(), nil

	// Enable action
	case CommandEnable:
		const enabled = true
		if len(args) < 3 {
			return fmt.Sprintf(actionNameMissing, a.ActionsTabularOutput()), nil
		}
		actionName := args[2]
		a.log.Debug("Enabling action...", actionName)

		if err := a.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
			return "", fmt.Errorf("while setting action %q to %t: %w", actionName, enabled, err)
		}

		return fmt.Sprintf(actionEnabled, actionName, clusterName), nil

	// Disable action
	case CommandDisable:
		const enabled = false
		if len(args) < 3 {
			return fmt.Sprintf(actionNameMissing, a.ActionsTabularOutput()), nil
		}
		actionName := args[2]
		a.log.Debug("Disabling action...", actionName)

		if err := a.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
			return "", fmt.Errorf("while setting action %q to %t: %w", actionName, enabled, err)
		}

		return fmt.Sprintf(actionDisabled, actionName, clusterName), nil
	}

	cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
	return "", errUnsupportedCommand
}

// ActionsTabularOutput sorts actions by key and returns a printable table
func (a *ActionExecutor) ActionsTabularOutput() string {
	// sort keys
	keys := make([]string, 0, len(a.actions))
	for k := range a.actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ACTION\tENABLED \tDISPLAY NAME")
	for _, name := range keys {
		fmt.Fprintf(w, "%s\t%v \t%s\n", name, a.actions[name].Enabled, a.actions[name].DisplayName)
	}
	w.Flush()
	return buf.String()
}
