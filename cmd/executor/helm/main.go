package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/executor/helm"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

const pluginName = "echo"

var urls = map[string]string{
	"darwin/amd64": "helm.sh/....",
}

func main() {
	hExec := helm.NewExecutor()

	runtime.GOOS
	runtime.GOARCH

	out, err := hExec.Execute(context.Background(), executor.ExecuteInput{
		//Command: "install --repo https://charts.bitnami.com/bitnami myspql postgresql --create-namespace -n botkube",
		Command: "install --repo https://charts.bitnami.com/bitnami postgresql --create-namespace -n test2 --generate-name --set clusterDomain='testing.local'",
	})

	// Botkube process:
	// 1. mkdir /tmp/plugin-name
	// 2. hExec.Initialize(config{installDir: "/tmp/plugin-name")

	if err != nil {
		panic(err)
	}
	fmt.Println(out.Data)
	return
	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: hExec,
		},
	})
}
