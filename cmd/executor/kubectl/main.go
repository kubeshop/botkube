package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/executor/kubectl"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	executor.Serve(map[string]plugin.Plugin{
		kubectl.PluginName: &executor.Plugin{
			Executor: kubectl.NewExecutor(version, kubectl.NewBinaryRunner()),
		},
	})
}
