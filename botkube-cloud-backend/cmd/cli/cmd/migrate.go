package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli/migrate"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli/printer"
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
			file, err := os.ReadFile(opts.ConfigFile)
			if err != nil {
				return err
			}

			instanceID, err := migrate.Run(cmd.Context(), status, file, opts)
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
	flags.StringVar(&opts.ConfigFile, "config-file", "./final-cfg.yaml", "Botkube deployment configuration")

	return login
}
