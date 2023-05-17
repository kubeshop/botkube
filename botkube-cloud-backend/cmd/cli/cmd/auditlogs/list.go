package auditlogs

import (
	"time"

	"github.com/spf13/cobra"
)

// ListOptions holds list related options.
type ListOptions struct {
	Output string
	Since  time.Duration
}

// NewList returns a new cobra.Command for listing available audit logs.
func NewList() *cobra.Command {
	var opts ListOptions

	cmd := &cobra.Command{
		Use:   "list [OPTIONS]",
		Short: "Prints recent audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Output, "output", "o", "table", "Output format. One of: json, yaml, table")
	flags.DurationVar(&opts.Since, "since", 0, "Only return logs newer than a relative duration like 5s, 2m, or 3h.")

	return cmd
}
