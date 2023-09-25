package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

// NewEnable returns a new cobra.Command for enabling telemetry.
func NewEnable() *cobra.Command {
	enable := &cobra.Command{
		Use:   "enable",
		Short: "Enable Botkube telemetry",
		Example: heredoc.WithCLIName(`
			# To improve the user experience, Botkube collects anonymized data.
			# It does not collect any identifying information, and all analytics
			# are used only as aggregated collection of data to improve Botkube
			# and adjust its roadmap.
			# Read our privacy policy at https://docs.botkube.io/privacy

			# Enable Botkube telemetry
			<cli> telemetry enable
		

		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			config := cli.NewConfig()
			config.Telemetry = cli.TelemetryEnabled
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
