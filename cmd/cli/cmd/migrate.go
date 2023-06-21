package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/kubeshop/botkube/internal/cli/migrate"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// NewMigrate returns a cobra.Command for migrate the OS into Cloud.
func NewMigrate() *cobra.Command {
	var opts migrate.Options

	login := &cobra.Command{
		Use:   "migrate [OPTIONS]",
		Short: "Automatically migrates Open Source installation into Botkube Cloud",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			status := printer.NewStatus(cmd.OutOrStdout(), "Migrating Botkube open source installation...")
			defer func() {
				status.End(err == nil)
			}()

			status.Step("Fetching Botkube configuration")
			cfg, err := migrate.GetConfigFromCluster(cmd.Context(), opts)
			if err != nil {
				return err
			}

			status.Step("Run Botkube migration")
			instanceID, err := migrate.Run(cmd.Context(), status, cfg, opts)
			if err != nil {
				return err
			}

			okCheck := color.New(color.FgGreen).FprintlnFunc()
			okCheck(cmd.OutOrStdout(), "\nMigration Succeeded 🎉")

			return browser.OpenURL(fmt.Sprintf("%s/instances/%s", opts.CloudDashboardURL, instanceID))
		},
	}

	flags := login.Flags()
	flags.StringVar(&opts.InstanceName, "instance-name", "", "Botkube Cloud Instance name that will be created")
	flags.StringVar(&opts.CloudAPIURL, "cloud-dashboard-url", "http://localhost:8080/graphql", "Botkube Cloud Instance name that will be created")
	flags.StringVar(&opts.CloudDashboardURL, "cloud-api-url", "http://localhost:3000", "Botkube Cloud Instance name that will be created")
	flags.StringVarP(&opts.Label, "label", "l", "app=botkube", "Label of botkube pod")
	flags.StringVarP(&opts.Namespace, "namespace", "n", "botkube", "Namespace of botkube pod")

	return login
}
