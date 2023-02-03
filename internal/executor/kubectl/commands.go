package kubectl

import (
	"fmt"
	"github.com/kubeshop/botkube/pkg/multierror"
	"strings"
)

// notSupportedSubcommands defines all explicitly not supported Kubectl plugin commands.
var notSupportedSubcommands = map[string]struct{}{
	"edit":         {},
	"attach":       {},
	"port-forward": {},
	"proxy":        {},
	"copy":         {},
	"debug":        {},
	"completion":   {},
}

var notSupportedGlobalFlags = map[string]struct{}{
	"--alsologtostderr":          {},
	"--as":                       {},
	"--as-group":                 {},
	"--as-uid":                   {},
	"--cache-dir":                {},
	"--certificate-authority":    {},
	"--client-certificate":       {},
	"--client-key":               {},
	"--insecure-skip-tls-verify": {},
	"--kubeconfig":               {},
	"--log-backtrace-at":         {},
	"--log-dir":                  {},
	"--log-file":                 {},
	"--log-file-max-size":        {},
	"--log-flush-frequency":      {},
	"--logtostderr":              {},
	"--token":                    {},
	"--user":                     {},
	"--username":                 {},
}

func normalizeCommand(command string) (string, error) {
	command = strings.TrimSpace(command)
	if !strings.HasPrefix(command, PluginName) {
		return "", fmt.Errorf("the input command does not target the %s plugin", PluginName)
	}
	command = strings.TrimPrefix(command, PluginName)

	return command, nil
}
func detectNotSupportedCommands(normalizedCmd string) error {
	args := strings.Fields(normalizedCmd)
	if len(args) <= 1 {
		return nil
	}

	_, found := notSupportedSubcommands[args[1]]
	if found {
		return fmt.Errorf("The %q command is not supported by the Botkube kubectl plugin.", args[1])
	}
	return nil
}

func detectNotSupportedGlobalFlags(normalizedCmd string) error {
	issues := multierror.New()
	for flagName := range notSupportedGlobalFlags {
		if !strings.Contains(normalizedCmd, flagName) {
			continue
		}
		issues = multierror.Append(issues, fmt.Errorf("The %q flag is not supported by the Botkube kubectl plugin. Please remove it.", flagName))
	}

	return issues.ErrorOrNil()
}
