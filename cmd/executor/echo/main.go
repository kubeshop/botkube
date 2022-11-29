package main

import (
	"context"
	"strings"

	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

const pluginName = "echo"

// Config holds executor configuration.
type Config struct {
	ChangeResponseToUpperCase bool
}

// EchoExecutor implements Botkube executor plugin.
type EchoExecutor struct{}

// Execute returns a given command as response.
func (EchoExecutor) Execute(_ context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// TODO(configure plugin): in request we should receive the executor configuration.
	cfg := Config{}

	data := req.Command
	if cfg.ChangeResponseToUpperCase {
		data = strings.ToUpper(data)
	}

	return &executor.ExecuteResponse{
		Data: data,
	}, nil
}

func main() {
	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: &EchoExecutor{},
		},
	})
}
