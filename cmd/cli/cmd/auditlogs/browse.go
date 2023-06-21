package auditlogs

import (
	"github.com/spf13/cobra"
)

// NewBrowse returns a cobra.Command for browsing Audit Logs.
func NewBrowse() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Interactively browse all audit logs in your terminal",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
