package telemetry

import "github.com/spf13/cobra"

// NewCmd returns a new cobra.Command subcommand for telemetry-related operations.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "telemetry",
		Short: "Configure collection of anonymous analytics",
	}

	root.AddCommand(
		NewEnable(),
		NewDisable(),
	)
	return root
}
