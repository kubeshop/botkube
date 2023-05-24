package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/executor/helm"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	executor.Serve(map[string]plugin.Plugin{
		helm.PluginName: &executor.Plugin{
			Executor: helm.NewExecutor(version),
		},
	})
}