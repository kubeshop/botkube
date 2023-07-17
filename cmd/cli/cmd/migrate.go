package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	semver "github.com/hashicorp/go-version"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"go.szostok.io/version"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/migrate"
	"github.com/kubeshop/botkube/internal/cli/printer"
)

const (
	botkubeVersionConstraints = ">= 1.0, < 1.3"

	containerName = "botkube"
)

var DefaultImageTag = "v9.99.9-dev"

// NewMigrate returns a cobra.Command for migrate the OS into Cloud.
func NewMigrate() *cobra.Command {
	var opts migrate.Options

	login := &cobra.Command{
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
		- RBAC is defaulted
		- Plugins are sourced from Botkube repository

		Use label selector to choose which Botkube pod you want to migrate. By default it's set to app=botkube.

		Examples:

            $ <cli> migrate --label app=botkube --instance-name botkube-slack     # Creates new Botkube Cloud instance with name botkube-slack and migrates pod with label app=botkube to it

			`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			status := printer.NewStatus(cmd.OutOrStdout(), "Migrating Botkube installation to Cloud")
			defer func() {
				status.End(err == nil)
			}()

			status.Step("Fetching Botkube configuration")
			cfg, pod, err := migrate.GetConfigFromCluster(cmd.Context(), opts)
			if err != nil {
				return err
			}

			botkubeVersionStr, err := getBotkubeVersion(pod)
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
			if !isCompatible {
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
			instanceID, err := migrate.Run(cmd.Context(), status, cfg, opts)
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

	flags := login.Flags()
	flags.StringVar(&opts.Token, "token", "", "Botkube Cloud authentication token")
	flags.StringVar(&opts.InstanceName, "instance-name", "", "Botkube Cloud Instance name that will be created")
	flags.StringVar(&opts.CloudAPIURL, "cloud-api-url", "https://api.botkube.io/graphql", "Botkube Cloud API URL")
	flags.StringVar(&opts.CloudDashboardURL, "cloud-dashboard-url", "https://app.botkube.io", "Botkube Cloud URL")
	flags.StringVarP(&opts.Label, "label", "l", "app=botkube", "Label of Botkube pod")
	flags.StringVarP(&opts.Namespace, "namespace", "n", "botkube", "Namespace of Botkube pod")
	flags.BoolVarP(&opts.SkipConnect, "skip-connect", "q", false, "Skips connecting to Botkube Cloud after migration")
	flags.BoolVar(&opts.SkipOpenBrowser, "skip-open-browser", false, "Skips opening web browser after migration")
	flags.BoolVarP(&opts.AutoApprove, "auto-approve", "y", false, "Skips interactive approval for upgrading Botkube installation.")
	flags.StringVar(&opts.ConfigExporter.Registry, "cfg-exporter-image-registry", "ghcr.io", "Config Exporter job image registry")
	flags.StringVar(&opts.ConfigExporter.Repository, "cfg-exporter-image-repo", "kubeshop/botkube-config-exporter", "Config Exporter job image repository")
	flags.StringVar(&opts.ConfigExporter.Tag, "cfg-exporter-image-tag", DefaultImageTag, "Config Exporter job image tag")
	flags.DurationVar(&opts.ConfigExporter.PollPeriod, "cfg-exporter-poll-period", 1*time.Second, "Config Exporter job poll period")
	flags.DurationVar(&opts.ConfigExporter.Timeout, "cfg-exporter-timeout", 1*time.Minute, "Config Exporter job timeout")

	return login
}

func getBotkubeVersion(p *corev1.Pod) (string, error) {
	for _, c := range p.Spec.Containers {
		if c.Name == containerName {
			fqin := strings.Split(c.Image, ":")
			if len(fqin) > 1 {
				return fqin[len(fqin)-1], nil
			}
			break
		}
	}
	return "", fmt.Errorf("unable to get botkube version: pod %q does not have botkube container", p.Name)
}
