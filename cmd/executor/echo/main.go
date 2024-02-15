package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/plugin"
)

var (
	// version is set via ldflags by GoReleaser.
	version = "dev"

	//go:embed config_schema.json
	configJSONSchema string
)

const (
	pluginName  = "echo"
	description = "Echo is an example Botkube executor plugin used during e2e tests. It's not meant for production usage."
)

// Config holds executor configuration.
type Config struct {
	ChangeResponseToUpperCase *bool `yaml:"changeResponseToUpperCase,omitempty"`
}

// EchoExecutor implements Botkube executor plugin.
type EchoExecutor struct{}

var _ executor.Executor = &EchoExecutor{}

// Metadata returns details about Echo plugin.
func (*EchoExecutor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     version,
		Description: description,
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
	}, nil
}

// Execute returns a given command as response.
func (*EchoExecutor) Execute(_ context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	var cfg Config
	err := plugin.MergeExecutorConfigs(in.Configs, &cfg)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configuration: %w", err)
	}

	data := in.Command
	if strings.Contains(data, "@fail") {
		return executor.ExecuteOutput{}, errors.New("The @fail label was specified. Failing execution.")
	}

	if strings.Contains(data, "@panic") {
		panic("The @panic label was specified. Panicking.")
	}

	if cfg.ChangeResponseToUpperCase != nil && *cfg.ChangeResponseToUpperCase {
		data = strings.ToUpper(data)
	}

	return executor.ExecuteOutput{
		Message: api.NewCodeBlockMessage(data, true),
	}, nil
}

// Help returns help message
func (*EchoExecutor) Help(context.Context) (api.Message, error) {
	return api.Message{}, nil
}

func main() {
	executor.Serve(map[string]goplugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: &EchoExecutor{},
		},
	})
}
