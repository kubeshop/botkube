package cmd

import (
	"github.com/spf13/cobra"
	"go.szostok.io/version/extension"

	"github.com/kubeshop/botkube/cmd/cli/cmd/bench"
	"github.com/kubeshop/botkube/cmd/cli/cmd/config"
	"github.com/kubeshop/botkube/cmd/cli/cmd/telemetry"
	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

const (
	orgName  = "kubeshop"
	repoName = "botkube"
)

// NewRoot returns a root cobra.Command for the whole Botkube Cloud CLI.
func NewRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   cli.Name,
		Short: "Botkube CLI",
		Long: heredoc.WithCLIName(`
        <cli> - Botkube CLI

        A utility that simplifies working with Botkube.

        Quick Start:

            $ <cli> install                              # Install Botkube
            $ <cli> uninstall                            # Uninstall Botkube

        Botkube Cloud:

            $ <cli> login                                # Login into Botkube Cloud
            $ <cli> migrate                              # Automatically migrates Open Source installation into Botkube Cloud
            `, cli.Name),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cli.RegisterVerboseModeFlag(rootCmd.PersistentFlags())

	rootCmd.AddCommand(
		NewLogin(),
		NewMigrate(),
		NewDocs(),
		NewInstall(),
		NewUninstall(),
		config.NewCmd(),
		bench.NewCmd(),
		telemetry.NewCmd(),
		extension.NewVersionCobraCmd(
			extension.WithUpgradeNotice(orgName, repoName),
		),
	)

	return rootCmd
}
