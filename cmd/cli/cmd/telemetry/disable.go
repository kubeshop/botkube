package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

// NewEnable returns a new cobra.Command for disabling telemetry.
func NewDisable() *cobra.Command {
	enable := &cobra.Command{
		Use:   "disable",
		Short: "Disable Botkube telemetry",
		Example: heredoc.WithCLIName(`
			# The Botkube CLI tool collects anonymous usage analytics.
			# This data is only available to the Botkube authors and helps us improve the tool.

			# Disable Botkube telemetry
			<cli> telemetry disable
		

		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			config := cli.NewConfig()
			config.Telemetry = "disabled"
			err = config.Save()
			if err != nil {
				return err
			}
			cmd.Println("Telemetry disabled")
			return nil
		},
	}

	return enable
}
