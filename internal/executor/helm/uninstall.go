package helm

import (
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// UninstallCommandAliases
type UninstallCommandAliases struct {
	Uninstall *UninstallCommand `arg:"subcommand:uninstall"`
	Un        *UninstallCommand `arg:"subcommand:un"`
	Delete    *UninstallCommand `arg:"subcommand:delete"`
	Del       *UninstallCommand `arg:"subcommand:del"`
}

func (u UninstallCommandAliases) Get() *UninstallCommand {
	if u.Uninstall != nil {
		return u.Uninstall
	}
	if u.Un != nil {
		return u.Un
	}
	if u.Delete != nil {
		return u.Delete
	}
	if u.Del != nil {
		return u.Del
	}

	return nil
}

// UninstallCommand holds possible uninstallation options such as positional arguments and supported flags
// Syntax:
//
//	helm uninstall RELEASE_NAME [...] [flags]
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

		Aliases:
		    uninstall, del, delete, un

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedUninstallFlags{}), 4))
}
