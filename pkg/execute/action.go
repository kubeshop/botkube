package execute

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/sirupsen/logrus"
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
	// sort keys
	keys := make([]string, 0, len(cfg.Actions))
	for k := range cfg.Actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	actions := make(map[string]config.Action)
	for _, name := range keys {
		actions[name] = cfg.Actions[name]
	}
	return &ActionExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfgManager:        cfgManager,
		actions:           actions,
	}
}

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
		a.analyticsReporter.ReportCommand(platform, cmdToReport, conversation.CommandOrigin, false)
	}()

	switch CommandVerb(cmdVerb) {
	case CommandList:
		a.log.Debug("List actions")
		return actionsTabularOutput(a.actions), nil

	// Enable action
	case CommandEnable:
		const enabled = true
		if len(args) < 3 {
			return fmt.Sprintf(actionNameMissing, actionsTabularOutput(a.actions)), nil
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
			return fmt.Sprintf(actionNameMissing, actionsTabularOutput(a.actions)), nil
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

func actionsTabularOutput(actions map[string]config.Action) string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ACTION\tENABLED \tDISPLAY NAME")
	for name, action := range actions {
		fmt.Fprintf(w, "%s\t%v \t%s\n", name, action.Enabled, action.DisplayName)
	}
	w.Flush()
	return buf.String()
}
