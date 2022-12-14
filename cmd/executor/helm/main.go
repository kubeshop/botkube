package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/executor/helm"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

const pluginName = "echo"


func main() {
	hExec := helm.NewExecutor()

	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: hExec,
		},
	})
}

