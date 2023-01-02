package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// HistoryCommandAliases holds different names for list subcommand.
// Unfortunately, it's a go-arg limitation that we cannot on a single entry have subcommand aliases.
type HistoryCommandAliases struct {
	History *HistoryCommand `arg:"subcommand:history"`
	Ls      *HistoryCommand `arg:"subcommand:hist"`
}

// Get returns HistoryCommand that were unpacked based on the alias used by user.
func (u HistoryCommandAliases) Get() *HistoryCommand {
	if u.History != nil {
		return u.History
	}
	if u.Ls != nil {
		return u.Ls
	}

	return nil
}

// HistoryCommand holds possible uninstallation options such as positional arguments and supported flags.
// Syntax:
//
//	helm history RELEASE_NAME [flags]
type HistoryCommand struct {
	noopValidator

	Name string `arg:"positional"`

	SupportedHistoryFlags
}

// Help returns command help message.
func (HistoryCommand) Help() string {
	return heredoc.Docf(`
		Shows historical revisions for a given release.

		A default maximum of 256 revisions will be returned. Setting '--max'
		configures the maximum length of the revision list returned.

		The historical release set is printed as a formatted table, e.g:

		    helm history angry-bird

		    REVISION    UPDATED                     STATUS          CHART             APP VERSION     DESCRIPTION
		    1           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Initial install
		    2           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Upgraded successfully
		    3           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Rolled back to 2
		    4           Mon Oct 3 10:15:13 2016     deployed        alpine-0.1.0      1.0             Upgraded successfully

		Usage:
		  helm history RELEASE_NAME [flags]

		Aliases:
		  history, hist

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedHistoryFlags{}), 4))
}

// SupportedHistoryFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedHistoryFlags struct {
	Max    int    `arg:"--max"`
	Output string `arg:"-o,--output"`
}
