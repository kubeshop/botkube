package main

import (
	"context"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubeshop/botkube/cmd/cli/cmd"
)

func main() {
	rootCmd := cmd.NewRoot()
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		// error is already printed by cobra, we can here add error switch
		// in case we would like to exit with different codes
		os.Exit(1)
	}
}
