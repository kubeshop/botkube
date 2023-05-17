package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/cmd/cli/cmd/auditlogs"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli/heredoc"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/loggerx"
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
            $ <cli> auditlogs list                       # Lists all audit logs from the default organization
            $ <cli> auditlogs list --organization 'foo'  # Lists all audit logs from the foo organization
            $ <cli> auditlogs list --interactive         # Interactively browse all audit logs in your terminal
            `, cli.Name),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			loggerx.ExitOnError(cmd.Help(), "while printing help")
		},
	}

	rootCmd.AddCommand(
		NewLogin(),
		NewMigrate(),
		auditlogs.NewCmd(),
	)

	return rootCmd
}
