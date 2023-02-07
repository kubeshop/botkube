package kubectl

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/multierror"
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
	"replace":      {},
	"wait":         {},
	"kustomize":    {},
}

// notSupportedGlobalFlags holds explicitly not supported flags in the format "<long>[,<short]". For example:
// - "add-dir-header" resolves to --add-dir-header
// - "server,s" resolves to --server and -s
var notSupportedGlobalFlags = map[string]struct{}{
	"add-dir-header":           {},
	"alsologtostderr":          {},
	"as":                       {},
	"as-group":                 {},
	"as-uid":                   {},
	"cache-dir":                {},
	"certificate-authority":    {},
	"client-certificate":       {},
	"client-key":               {},
	"cluster":                  {},
	"context":                  {},
	"insecure-skip-tls-verify": {},
	"kubeconfig":               {},
	"log-backtrace-at":         {},
	"log-dir":                  {},
	"log-file":                 {},
	"log-file-max-size":        {},
	"log-flush-frequency":      {},
	"logtostderr":              {},
	"one-output":               {},
	"password":                 {},
	"profile":                  {},
	"profile-output":           {},
	"server,s":                 {},
	"token":                    {},
	"skip-log-headers":         {},
	"stderrthreshold":          {},
	"tls-server-name":          {},
	"vmodule":                  {},
	"user":                     {},
	"username":                 {},
}

func normalizeCommand(command string) (string, error) {
	command = strings.TrimSpace(command)
	if !strings.HasPrefix(command, PluginName) {
		return "", fmt.Errorf("the input command does not target the %s plugin", PluginName)
	}
	command = strings.TrimPrefix(command, PluginName)

	return strings.TrimSpace(command), nil
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

	f := pflag.NewFlagSet("detect-not-supported-flags", pflag.ContinueOnError)
	f.ParseErrorsWhitelist.UnknownFlags = true

	for key := range notSupportedGlobalFlags {
		long, short, found := strings.Cut(key, ",")
		if found {
			f.StringP(long, short, "", "")
			continue
		}
		f.String(long, "", "")
	}

	err := f.Parse(strings.Fields(normalizedCmd))
	if err != nil {
		return fmt.Errorf("while parsing args: %w", err)
	}

	// visit ONLY flags which have been defined by f.String and explicitly set in the command:
	f.Visit(func(f *pflag.Flag) {
		if f == nil {
			return
		}
		issues = multierror.Append(issues, fmt.Errorf("The %q flag is not supported by the Botkube kubectl plugin. Please remove it.", f.Name))
	})

	return issues.ErrorOrNil()
}
