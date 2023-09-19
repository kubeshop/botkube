package cmd

import (
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	semver "github.com/hashicorp/go-version"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"go.szostok.io/version"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/analytics"
	"github.com/kubeshop/botkube/internal/cli/config"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/migrate"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
)

const (
	botkubeVersionConstraints = ">= 1.0, < 1.3"
)

// NewMigrate returns a cobra.Command for migrate the OS into Cloud.
func NewMigrate() *cobra.Command {
	var opts migrate.Options

	migrate := &cobra.Command{
		Use:   "migrate [OPTIONS]",
		Short: "Automatically migrates Botkube installation into Botkube Cloud",
		Long: heredoc.WithCLIName(`
		Automatically migrates Botkube installation to Botkube Cloud.
		This command will create a new Botkube Cloud instance based on your existing Botkube configuration, and upgrade your Botkube installation to use the remote configuration.
		
		Supported Botkube bot platforms for migration:
		- Socket Slack
		- Discord
		- Mattermost
		
		Limitations:
		- Plugins are sourced from Botkube repository

		Use label selector to choose which Botkube pod you want to migrate. By default it's set to app=botkube.

		Examples:

            $ <cli> migrate --label app=botkube --instance-name botkube-slack     # Creates new Botkube Cloud instance with name botkube-slack and migrates pod with label app=botkube to it

			`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			k8sConfig, err := kubex.LoadRestConfigWithMetaInformation()
			if err != nil {
				return fmt.Errorf("while creating k8s config: %w", err)
			}

			status := printer.NewStatus(cmd.OutOrStdout(), "Migrating Botkube installation to Cloud")
			defer func() {
				status.End(err == nil)
			}()

			cfg, botkubeVersionStr, err := config.GetFromCluster(cmd.Context(), status, k8sConfig.K8s, opts.ConfigExporter, opts.AutoApprove)
			if err != nil {
				return err
			}

			status.Infof("Checking if Botkube version %q can be migrated safely", botkubeVersionStr)

			constraint, err := semver.NewConstraint(botkubeVersionConstraints)
			if err != nil {
				return fmt.Errorf("unable to parse Botkube semver version constraints: %w", err)
			}

			botkubeVersion, err := semver.NewVersion(botkubeVersionStr)
			if err != nil {
				return fmt.Errorf("unable to parse botkube version %s as semver: %w", botkubeVersion, err)
			}

			isCompatible := constraint.Check(botkubeVersion)
			if !isCompatible && !opts.AutoApprove {
				run := false

				prompt := &survey.Confirm{
					Message: heredoc.Docf(`
						
						The migration process for the Botkube CLI you're using (version: %q) wasn't tested with your Botkube version on your cluster (%q).
						Botkube version constraints for the currently installed CLI: %s
						We recommend upgrading your CLI to the latest version. In order to do so, navigate to https://docs.botkube.io/.
						
						Do you wish to continue?`, version.Get().Version, botkubeVersion, botkubeVersionConstraints),
					Default: false,
				}

				err = survey.AskOne(prompt, &run)
				if err != nil {
					return err
				}
				if !run {
					status.Infof("Aborting migration.")
					return nil
				}
			}

			status.Step("Run Botkube migration")
			instanceID, err := migrate.Run(cmd.Context(), status, cfg, k8sConfig, opts)
			if err != nil {
				return err
			}

			okCheck := color.New(color.FgGreen).FprintlnFunc()
			okCheck(cmd.OutOrStdout(), "\nMigration Succeeded 🎉")

			instanceURL := fmt.Sprintf("%s/instances/%s", opts.CloudDashboardURL, instanceID)

			if opts.SkipOpenBrowser {
				fmt.Println(heredoc.Docf(`
				 Visit the URL to see your instance details:
				 %s
		`, instanceURL))
				return nil
			}

			fmt.Println(heredoc.Docf(`
			If your browser didn't open automatically, visit the URL to see your instance details:
				 %s
		`, instanceURL))
			return browser.OpenURL(instanceURL)
		},
	}

	migrate = analytics.InjectAnalyticsReporting(*migrate, "migrate")

	flags := migrate.Flags()

	flags.Bool(analytics.OptOutAnalyticsFlag, false, analytics.OptOutAnalyticsFlagUsage)
	flags.DurationVar(&opts.Timeout, "timeout", 10*time.Minute, `Maximum time during which the Botkube installation is being watched, where "0" means "infinite". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
	flags.StringVar(&opts.Token, "token", "", "Botkube Cloud authentication token")
	flags.BoolVarP(&opts.Watch, "watch", "w", true, "Watches the status of the Botkube installation until it finish or the defined `--timeout` occurs.")
	flags.StringVar(&opts.InstanceName, "instance-name", "", "Botkube Cloud Instance name that will be created")
	flags.StringVar(&opts.CloudAPIURL, "cloud-api-url", "https://api.botkube.io/graphql", "Botkube Cloud API URL")
	flags.StringVar(&opts.CloudDashboardURL, "cloud-dashboard-url", "https://app.botkube.io", "Botkube Cloud URL")
	flags.BoolVarP(&opts.SkipConnect, "skip-connect", "q", false, "Skips connecting to Botkube Cloud after migration")
	flags.BoolVar(&opts.SkipOpenBrowser, "skip-open-browser", false, "Skips opening web browser after migration")
	flags.BoolVarP(&opts.AutoApprove, "auto-approve", "y", false, "Skips interactive approval for upgrading Botkube installation.")
	flags.StringVarP(&opts.ImageTag, "image-tag", "", "", "Botkube image tag, possible values latest, v1.2.0, ...")

	opts.ConfigExporter.RegisterFlags(flags)
	kubex.RegisterKubeconfigFlag(flags)

	return migrate
}
