package cmd

import (
	"context"
	"io"
	"os"

	"github.com/fatih/color"

	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli/heredoc"
)

// NewLogin returns a cobra.Command for logging into a Botkube Cloud.
func NewLogin() *cobra.Command {
	login := &cobra.Command{
		Use:   "login [OPTIONS]",
		Short: "Login to a Botkube Cloud",
		Example: heredoc.WithCLIName(`
			# start interactive setup
			<cli> login
		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), os.Stdout)
		},
	}

	return login
}

func runLogin(_ context.Context, w io.Writer) error {
	okCheck := color.New(color.FgGreen).FprintlnFunc()
	okCheck(w, "Login Succeeded\n")

	return nil
}
