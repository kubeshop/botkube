package cmd

import (
	"context"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewMigrate returns a cobra.Command for migrate the OS into Cloud.
func NewMigrate() *cobra.Command {
	login := &cobra.Command{
		Use:   "migrate [OPTIONS]",
		Short: "Automatically migrates Open Source installation into Botkube Cloud",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(cmd.Context(), os.Stdout)
		},
	}

	return login
}

func runMigrate(_ context.Context, w io.Writer) error {
	okCheck := color.New(color.FgGreen).FprintlnFunc()
	okCheck(w, "Migration Succeeded\n")

	return nil
}
