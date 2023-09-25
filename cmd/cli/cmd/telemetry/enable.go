package telemetry

import (
	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"

	"github.com/spf13/cobra"
)

// NewEnable returns a new cobra.Command for enabling telemetry.
func NewEnable() *cobra.Command {
	enable := &cobra.Command{
		Use:   "enable",
		Short: "Enable Botkube telemetry",
		Example: heredoc.WithCLIName(`
			# The Botkube CLI tool collects anonymous usage analytics.
			# This data is only available to the Botkube authors and helps us improve the tool.

			# Enable Botkube telemetry
			<cli> telemetry enable
		

		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			config := cli.NewConfig()
			config.Telemetry = "enabled"
			err = config.Save()
			if err != nil {
				return err
			}
			cmd.Println("Telemetry enabled")
			return nil
		},
	}

	return enable
}
