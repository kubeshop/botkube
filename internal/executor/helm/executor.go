package helm

import (
	"context"
	"fmt"

	"github.com/alexflint/go-arg"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

const (
	// PluginName is the name of the Helm Botkube plugin.
	PluginName       = "helm"
	helmBinaryName   = "helm"
	defaultNamespace = "default"
)

type command interface {
	Validate() error
	Help() string
}

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
		Description: "Helm is the Botkube executor plugin that allows you to run the Helm CLI commands directly from any communication platform.",
	}, nil
}

// Execute returns a given command as response.
//
// Supported commands:
// - install
// - uninstall
// - list
// - version
// - status
// - test
// - rollback
// - upgrade
// - history
// - get [all|manifest|hooks|notes]
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

	if helmCmd.Namespace == "" { // use 'default' namespace, instead of namespace where botkube was installed
		args = append([]string{"-n", defaultNamespace}, args...)
	}

	switch {
	case helmCmd.Install != nil:
		return e.handleHelmCommand(ctx, helmCmd.Install, cfg, wasHelpRequested, args)
	case helmCmd.UninstallCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.UninstallCommandAliases.Get(), cfg, wasHelpRequested, args)
	case helmCmd.ListCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.ListCommandAliases.Get(), cfg, wasHelpRequested, args)
	case helmCmd.Version != nil:
		return e.handleHelmCommand(ctx, helmCmd.Version, cfg, wasHelpRequested, args)
	case helmCmd.Status != nil:
		return e.handleHelmCommand(ctx, helmCmd.Status, cfg, wasHelpRequested, args)
	case helmCmd.Test != nil:
		return e.handleHelmCommand(ctx, helmCmd.Test, cfg, wasHelpRequested, args)
	case helmCmd.Rollback != nil:
		return e.handleHelmCommand(ctx, helmCmd.Rollback, cfg, wasHelpRequested, args)
	case helmCmd.Upgrade != nil:
		return e.handleHelmCommand(ctx, helmCmd.Upgrade, cfg, wasHelpRequested, args)
	case helmCmd.HistoryCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.HistoryCommandAliases.Get(), cfg, wasHelpRequested, args)
	case helmCmd.Get != nil:
		switch {
		case helmCmd.Get.All != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.All, cfg, wasHelpRequested, args)
		case helmCmd.Get.Hooks != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Hooks, cfg, wasHelpRequested, args)
		case helmCmd.Get.Manifest != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Manifest, cfg, wasHelpRequested, args)
		case helmCmd.Get.Notes != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Notes, cfg, wasHelpRequested, args)
		case helmCmd.Get.Values != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Values, cfg, wasHelpRequested, args)
		default:
			return executor.ExecuteOutput{
				Data: helmCmd.Get.Help(),
			}, nil
		}
	case helmCmd.Help != nil, wasHelpRequested:
		return executor.ExecuteOutput{
			Data: helmCmd.Help.Help(),
		}, nil
	default:
		return executor.ExecuteOutput{
			Data: "Helm command not supported",
		}, nil
	}
}

// handleHelmList construct a Helm CLI command and run it.
func (e *Executor) handleHelmCommand(ctx context.Context, cmd command, cfg Config, wasHelpRequested bool, args []string) (executor.ExecuteOutput, error) {
	if wasHelpRequested {
		return executor.ExecuteOutput{
			Data: cmd.Help(),
		}, nil
	}

	err := cmd.Validate()
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	out, err := e.runHelmCLIBinary(ctx, cfg, args)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("%s\n%s", out, err.Error())
	}

	return executor.ExecuteOutput{
		Data: out,
	}, nil
}
