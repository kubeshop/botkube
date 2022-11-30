package main

import (
	"context"
	"strings"

	"github.com/hashicorp/go-plugin"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

const pluginName = "echo"

// Config holds executor configuration.
type Config struct {
	ChangeResponseToUpperCase *bool `yaml:"changeResponseToUpperCase,omitempty"`
}

// EchoExecutor implements Botkube executor plugin.
type EchoExecutor struct{}

// Execute returns a given command as response.
func (EchoExecutor) Execute(_ context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// In our case we don't have complex merge strategy,
	// the last one that was specified wins :)
	finalCfg := Config{}
	for _, rawCfg := range req.Configs {
		var cfg Config
		err := yaml.Unmarshal(rawCfg, &cfg)
		if err != nil {
			return nil, err
		}
		if cfg.ChangeResponseToUpperCase == nil {
			continue
		}
		finalCfg.ChangeResponseToUpperCase = cfg.ChangeResponseToUpperCase
	}

	data := req.Command
	if finalCfg.ChangeResponseToUpperCase != nil && *finalCfg.ChangeResponseToUpperCase {
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
