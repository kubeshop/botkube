package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// HelpCommand holds possible help options such as positional arguments and supported flags.
// Syntax:
//
//	helm help
type HelpCommand struct{}

// Help returns command help message.
func (*HelpCommand) Help() string {
	return heredoc.Docf(`
		The official Botkube plugin for the Helm CLI.

		Usage:
		  helm [command]

		Available Commands:
		  install     # Installs a given chart to cluster where Botkube is installed.
		  list        # Lists all releases on cluster where Botkube is installed.
		  rollback    # Rolls back a given release to a previous revision.
		  status      # Displays the status of the named release.
		  test        # Runs tests for a given release.
		  uninstall   # Uninstalls a given release.
		  upgrade     # Upgrades a given release.
		  version     # Shows the version of the Helm CLI used by this Botkube plugin.
		  history     # Shows release history
		  get         # Shows extended information of a named release

		Flags:
		%s

		Use "helm [command] --help" for more information about the command.
	`, indent.String(renderSupportedFlags(GlobalFlags{}), 4))
}
