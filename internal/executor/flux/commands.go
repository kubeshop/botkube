package flux

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// deleteConfirmPhase represent a confirmation phase for deletion. Taken from flux: v2.0.1.
const deleteConfirmPhase = "Are you sure you want to delete"

var deleteConfirmErr = errors.New("To delete the resource, please explicitly include the -s or --silent flag in your command")

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
	opts = append(opts, pluginx.ExecuteClearColorCodes())
	out, err := pluginx.ExecuteCommand(ctx, in, opts...)
	if err != nil {
		return "", err
	}
	return out.CombinedOutput(), nil
}

// isDeleteConfirmationErr uses string contains in order to detect if a user was asked to confirm deletion.
// For now, there is no better way as we use terminal output not Go SDK.
func isDeleteConfirmationErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), deleteConfirmPhase)
}
