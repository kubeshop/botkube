package helm

import (
	"context"
	"fmt"

	"github.com/alexflint/go-arg"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

const (
	helmBinaryName = "helm"

	// PluginName is the name of the Helm Botkube plugin.
	PluginName = "helm"
)

var _ executor.Executor = &Executor{}

// Executor provides functionality for running Helm CLI.
type Executor struct {
	pluginVersion    string
	runHelmCLIBinary func(ctx context.Context, cfg Config, args []string) (string, error)
}

// NewExecutor returns a new Executor instance.
func NewExecutor(ver string) *Executor {
	return &Executor{
		pluginVersion:    ver,
		runHelmCLIBinary: runHelmCLIBinary,
	}
}

// Metadata returns details about Helm plugin.
func (e *Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     e.pluginVersion,
		Description: "TBD",
	}, nil
}

// Execute returns a given command as response.
//
// Supported commands:
// - install
//
// TODO:
// - uninstall
// - upgrade
// - rollback
// - list
// - version
// - test
// - status
func (e *Executor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	cfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	var wasHelpRequested bool
	helmCmd, args, err := parseRawCommand(in.Command)
	switch err {
	case nil, arg.ErrVersion:
		// we ignore the --version flag, as Helm CLI will handle that.
	case arg.ErrHelp:
		// we want to print our own help instead of delegating that to Helm CLI.
		wasHelpRequested = true
	default:
		return executor.ExecuteOutput{}, fmt.Errorf("while parsing input command: %w", err)
	}

	switch {
	case helmCmd.Install != nil:
		return e.handleHelmInstall(ctx, cfg, wasHelpRequested, helmCmd.Install, args)
	default:
		return executor.ExecuteOutput{
			Data: "Command not supported",
		}, nil
	}
}

// handleHelmInstall construct a Helm CLI command and run it.
// Supported options:
//  1. By absolute URL:
//     helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
//  2. By chart reference and repo url:
//     helm install --repo https://example.com/charts/ mynginx nginx
func (e *Executor) handleHelmInstall(ctx context.Context, cfg Config, wasHelpRequested bool, install *InstallCommand, args []string) (executor.ExecuteOutput, error) {
	if wasHelpRequested {
		return executor.ExecuteOutput{
			Data: helpInstall(),
		}, nil
	}

	err := install.Validate()
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	out, err := e.runHelmCLIBinary(ctx, cfg, args)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("%s: %s", err.Error(), out)
	}

	return executor.ExecuteOutput{
		Data: out,
	}, nil
}
