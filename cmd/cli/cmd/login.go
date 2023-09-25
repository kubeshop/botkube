package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/analytics"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/login"
)

// NewLogin returns a cobra.Command for logging into a Botkube Cloud.
func NewLogin() *cobra.Command {
	var opts login.Options

	login := &cobra.Command{
		Use:   "login [OPTIONS]",
		Short: "Login to a Botkube Cloud",
		Example: heredoc.WithCLIName(`
			# start interactive setup
			<cli> login
		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return login.Run(cmd.Context(), os.Stdout, opts)
		},
	}

	login = analytics.InjectAnalyticsReporting(*login, "login")

	flags := login.Flags()
	flags.StringVar(&opts.CloudDashboardURL, "cloud-dashboard-url", "https://app.botkube.io", "Botkube Cloud URL")
	flags.StringVar(&opts.LocalServerAddress, "local-server-addr", "localhost:8085", "Address of a local server which is used for the login flow")

	return login
}
