package auditlogs

import (
	"github.com/spf13/cobra"
)

// NewCmd returns a new cobra.Command subcommand for Audit Logs related operations.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "auditlogs",
		Aliases: []string{"audit"},
		Short:   "This command consists of multiple subcommands for Audit Logs",
	}

	root.AddCommand(
		NewList(),
		NewBrowse(),
	)
	return root
}
