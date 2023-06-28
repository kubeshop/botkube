package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/migrate"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/pkg/browser"
)

var (
	compatibleBotkubeVersions = map[string]bool{
		"v1.0.0": true,
		"v1.0.1": true,
		"v1.1.0": true,
		"v1.2.0": true,
	}
)

// NewMigrate returns a cobra.Command for migrate the OS into Cloud.
func NewMigrate() *cobra.Command {
	var opts migrate.Options

	login := &cobra.Command{
		Use:   "migrate [OPTIONS]",
		Short: "Automatically migrates Botkube installation into Botkube Cloud",
		Long: heredoc.WithCLIName(`
		Automatically migrates Botkube installation into Botkube Cloud
		
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
			status := printer.NewStatus(cmd.OutOrStdout(), "Migrating Botkube open source installation...")
			defer func() {
				status.End(err == nil)
			}()

			status.Step("Fetching Botkube configuration")
			cfg, pod, err := migrate.GetConfigFromCluster(cmd.Context(), opts)
			if err != nil {
				return err
			}

			version, err := getBotkubeVersion(pod)
			if err != nil {
				return err
			}
			status.Infof("Checking Botkube version %q compatibility", version)
			if !compatibleBotkubeVersions[version] {
				run := false
				suportedVersions := strings.Join(maps.Keys(compatibleBotkubeVersions), ", ")
				prompt := &survey.Confirm{
					Message: fmt.Sprintf("Your Botkube version %q is not supported, migration might fail. Do you wish to continue?\nSupported versions: %s", version, suportedVersions),
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

			if opts.SkipConnect {
				return nil
			}
			return browser.OpenURL(fmt.Sprintf("%s/instances/%s", opts.CloudDashboardURL, instanceID))
		},
	}

	flags := login.Flags()
	flags.StringVar(&opts.Token, "token", "", "Botkube Cloud authentication token")
	flags.StringVar(&opts.InstanceName, "instance-name", "", "Botkube Cloud Instance name that will be created")
	flags.StringVar(&opts.CloudAPIURL, "cloud-api-url", "https://api.botkube.io", "Botkube Cloud API URL")
	flags.StringVar(&opts.CloudDashboardURL, "cloud-dashboard-url", "https://app.botkube.io", "Botkube Cloud URL")
	flags.StringVarP(&opts.Label, "label", "l", "app=botkube", "Label of Botkube pod")
	flags.StringVarP(&opts.Namespace, "namespace", "n", "botkube", "Namespace of Botkube pod")
	flags.BoolVarP(&opts.SkipConnect, "skip-connect", "q", false, "Skips connecting to Botkube Cloud after migration")

	return login
}

func getBotkubeVersion(p *corev1.Pod) (string, error) {
	for _, c := range p.Spec.Containers {
		if c.Name == "botkube" {
			fqin := strings.Split(c.Image, ":")
			if len(fqin) > 1 {
				return fqin[len(fqin)-1], nil
			}
			break
		}
	}
	return "", fmt.Errorf("unable to get botkube version: pod %q does not have botkube container", p.Name)
}
