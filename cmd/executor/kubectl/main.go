package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/executor/kubectl"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	kcRunner := kubectl.NewBinaryRunner()
	l := loggerx.New(loggerx.Config{
		Level:         "info",
		DisableColors: false,
	})
	executor.Serve(map[string]plugin.Plugin{
		kubectl.PluginName: &executor.Plugin{
			Executor: kubectl.NewExecutor(l, version, kcRunner),
		},
	})
}
