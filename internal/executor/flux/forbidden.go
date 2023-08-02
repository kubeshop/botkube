package flux

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/multierror"
)

// notSupportedGlobalFlags holds explicitly not supported flags in the format "<long>[,<short]". For example:
// - "add-dir-header" resolves to --add-dir-header
// - "server,s" resolves to --server and -s
var notSupportedGlobalFlags = map[string]struct{}{
	"as":                    {},
	"as-group":              {},
	"as-uid":                {},
	"certificate-authority": {},
	"client-certificate":    {},
	"client-key":            {},
	"cluster":               {},
	"context":               {},
	"kubeconfig":            {},
	"server":                {},
	"tls-server-name":       {},
	"user":                  {},
	"watch,w":               {},
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
		issues = multierror.Append(issues, fmt.Errorf("The %q flag is not supported by the Botkube flux plugin. Please remove it.", f.Name))
	})

	switch issues.Len() {
	case 0:
		return nil
	case 1:
		return issues.Errors[0]
	default:
		return issues.ErrorOrNil()
	}
}
