package telemetry

import "github.com/spf13/cobra"

// NewCmd returns a new cobra.Command subcommand for telemetry-related operations.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "telemetry",
		Short: "This command consists of subcommands to disable or enable telemetry",
	}

	root.AddCommand(
		NewEnable(),
		NewDisable(),
	)
	return root
}
