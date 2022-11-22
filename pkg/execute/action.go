package execute

import (
	"bytes"
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/kubeshop/botkube/pkg/config"
)

func (e *DefaultExecutor) runActionCommand(ctx context.Context, args []string, clusterName string) (string, error) {
	if len(args) < 2 {
		return "", errInvalidCommand
	}

	validSubCmd := false
	for _, cmdName := range []string{"action", "actions", "act"} {
		if args[1] == cmdName {
			validSubCmd = true
			break
		}
	}
	if !validSubCmd {
		return fmt.Sprintf("'%s' is not a valid subcommand", args[1]), nil
	}

	var cmdVerb = args[0]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", args[1], cmdVerb)
		e.reportCommand(cmdToReport, false)
	}()

	actions, err := e.cfgManager.ListActions(ctx)
	if err != nil {
		return "Failed to list actions", err
	}

	switch CommandVerb(cmdVerb) {
	case CommandList:
		e.log.Debug("List actions")
		return actionsTabularOutput(actions), nil

	// Enable action
	case CommandEnable:
		const enabled = true
		if len(args) < 3 {
			return fmt.Sprintf(actionNameMissing, actionsTabularOutput(actions)), nil
		}
		actionName := args[2]
		e.log.Debug("Enabling action...", actionName)

		if err := e.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
			return "", fmt.Errorf("while setting action %q to %t: %w", actionName, enabled, err)
		}

		return fmt.Sprintf(actionEnabled, actionName, clusterName), nil

	// Disable action
	case CommandDisable:
		const enabled = false
		if len(args) < 3 {
			return fmt.Sprintf(actionNameMissing, actionsTabularOutput(actions)), nil
		}
		actionName := args[2]
		e.log.Debug("Disabling action...", actionName)

		if err := e.cfgManager.PersistActionEnabled(ctx, actionName, enabled); err != nil {
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
