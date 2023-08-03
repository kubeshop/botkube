package flux

import (
	"context"
	"fmt"
	"strings"

	"github.com/gookit/color"

	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// escapePositionals add '--' after known keyword. Example:
//
//	old: `flux gh pr view 2`
//	new: `flux gh -- pr view 2`
//
// As a result, we can parse it without fully defining what 'gh' command is specified.
func escapePositionals(in, name string) string {
	if strings.Contains(in, name) {
		return strings.Replace(in, name, fmt.Sprintf("%s -- ", name), 1)
	}
	return in
}

func normalize(in string) string {
	out := formatx.RemoveHyperlinks(in)
	out = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(out)

	out = strings.TrimSpace(out)

	return out
}

// ExecuteCommand is a syntax sugar for running CLI commands.
func ExecuteCommand(ctx context.Context, in string, opts ...pluginx.ExecuteCommandMutation) (string, error) {
	out, err := pluginx.ExecuteCommand(ctx, in, opts...)
	if err != nil {
		return "", err
	}
	return color.ClearCode(out.CombinedOutput()), nil
}
