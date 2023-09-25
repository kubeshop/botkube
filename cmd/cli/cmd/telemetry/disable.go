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
			# To improve the user experience, Botkube collects anonymized data.
			# It does not collect any identifying information, and all analytics
			# are used only as aggregated collection of data to improve Botkube
			# and adjust its roadmap.
			# Read our privacy policy at https://docs.botkube.io/privacy

			# The Botkube CLI tool collects anonymous usage analytics.
			# This data is only available to the Botkube authors and helps us improve the tool.

			# Disable Botkube telemetry
			<cli> telemetry disable
		

		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			config := cli.NewConfig()
			config.Telemetry = cli.TelemetryDisabled
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
