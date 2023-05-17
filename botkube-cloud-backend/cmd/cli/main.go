package main

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/cmd/cli/cmd"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/loggerx"
)

func main() {
	rootCmd := cmd.NewRoot()
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	err := rootCmd.ExecuteContext(ctx)
	loggerx.ExitOnError(err, "while running CLI")
}
