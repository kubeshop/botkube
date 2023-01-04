package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// StatusCommand holds possible status options such as positional arguments and supported flags.
// Syntax:
//
//	helm status RELEASE_NAME [flags]
type StatusCommand struct {
	noopValidator

	Name string `arg:"positional"`

	SupportedStatusFlags
}

// Help returns command help message.
func (StatusCommand) Help() string {
	return heredoc.Docf(`
		Shows the status of a named release.

		The status consists of:
		- last deployment time
		- k8s namespace in which the release lives
		- state of the release (can be: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback)
		- revision of the release
		- description of the release (can be completion message or error message, need to enable --show-desc)
		- list of resources that this release consists of, sorted by kind
		- details on last test suite run, if applicable
		- additional notes provided by the chart

		Usage:
		  helm status RELEASE_NAME [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedStatusFlags{}), 4))
}

// SupportedStatusFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedStatusFlags struct {
	ShowDesc bool   `arg:"--show-desc"`
	Revision int    `arg:"--revision"`
	Output   string `arg:"--output"`
}
