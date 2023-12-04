package config

import (
	"github.com/spf13/cobra"
)

// NewCmd returns a new cobra.Command subcommand for config-related operations.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "This command consists of multiple subcommands for working with Botkube configuration",
	}

	root.AddCommand(
		NewGet(),
	)
	return root
}
