package helm

import (
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// UninstallCommand holds possible uninstallation options such as positional arguments and supported flags
// Syntax:
//
//	helm uninstall RELEASE_NAME [...] [flags]
//
// TODO:
//
//	uninstall, del, delete, un
type UninstallCommand struct {
	Name []string `arg:"positional"`

	SupportedUninstallFlags
	NotSupportedUninstallFlags
}

// Validate validates that all uninstallation parameters are valid.
func (i UninstallCommand) Validate() error {
	return returnErrorOfAllSetFlags(i.NotSupportedUninstallFlags)
}

// SupportedUninstallFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedUninstallFlags struct {
	Description string        `arg:"--description"`
	DryRun      bool          `arg:"--dry-run"`
	KeepHistory bool          `arg:"--keep-history"`
	NoHooks     bool          `arg:"--no-hooks"`
	Timeout     time.Duration `arg:"--timeout"`
}

// NotSupportedUninstallFlags represents flags supported by Helm CLI but not by Helm Plugin.
type NotSupportedUninstallFlags struct {
	Wait bool `arg:"--wait"`
}

func helpUninstall() string {
	return heredoc.Docf(`
		This command takes a release name and uninstalls the release.

		It removes all of the resources associated with the last release of the chart
		as well as the release history, freeing it up for future use.

		Usage:
		    helm uninstall RELEASE_NAME [...] [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedUninstallFlags{}), 4))
}
