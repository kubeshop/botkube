package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

// NewRoot returns a root cobra.Command for the whole Botkube Cloud CLI.
func NewRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   cli.Name,
		Short: "Botkube Cloud CLI",
		Long: heredoc.WithCLIName(`
        <cli> - Botkube Cloud CLI

        A utility that manages Botkube Cloud resources.

        To begin working with Botkube Cloud using the <cli> CLI, start with:

            $ <cli> login

        Quick Start:

            $ <cli> migrate                              # Automatically migrates Open Source installation into Botkube Cloud
            `, cli.Name),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(
		NewLogin(),
		NewMigrate(),
		NewDocs(),
	)

	return rootCmd
}
