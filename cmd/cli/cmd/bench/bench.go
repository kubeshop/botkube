package bench

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use: "bench",
	}

	root.AddCommand(
		NewGRPC(),
	)
	return root
}
