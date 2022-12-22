package helm

import (
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// RollbackCommand holds possible rollback options such as positional arguments and supported flags.
// Syntax:
//
//	helm RELEASE [REVISION] [flags]
type RollbackCommand struct {
	Name     string `arg:"positional"`
	Revision string `arg:"positional"`

	SupportedRollbackFlags
	NotSupportedRollbackFlags
}

// Validate validates that all list parameters are valid.
func (i RollbackCommand) Validate() error {
	return returnErrorOfAllSetFlags(i.NotSupportedRollbackFlags)
}

// Help returns command help message.
func (RollbackCommand) Help() string {
	return heredoc.Docf(`
		This command rolls back a release to a previous revision.

		The first argument of the rollback command is the name of a release, and the
		second is a revision (version) number. If this argument is omitted, it will
		roll back to the previous release.

		To see revision numbers, run 'helm history RELEASE'.

		Usage:
		  helm rollback RELEASE [REVISION] [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedRollbackFlags{}), 4))
}

// SupportedRollbackFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedRollbackFlags struct {
	CleanupOnFail bool          `arg:"--cleanup-on-fail"`
	DryRun        bool          `arg:"--dry-run"`
	Force         bool          `arg:"--force"`
	HistoryMax    int           `arg:"--history-max"`
	NoHooks       bool          `arg:"--no-hooks"`
	RecreatePods  bool          `arg:"--recreate-pods"`
	Timeout       time.Duration `arg:"--timeout"`
}

// NotSupportedRollbackFlags represents flags supported by Helm CLI but not by Helm Plugin.
type NotSupportedRollbackFlags struct {
	Wait        bool `arg:"--wait"`
	WaitForJobs bool `arg:"--wait-for-jobs"`
}
