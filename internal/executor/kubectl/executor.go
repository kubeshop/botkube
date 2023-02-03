package kubectl

import (
	"context"
	"fmt"
	"github.com/gookit/color"
	"os"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	// PluginName is the name of the Helm Botkube plugin.
	PluginName       = "kubectl"
	kcBinaryName     = "kubectl"
	defaultNamespace = "default"
	description      = "Helm is the Botkube executor plugin that allows you to run the Helm CLI commands directly from any communication platform."
)

// Links source: https://github.com/helm/helm/releases/tag/v3.6.3
// Using go-getter syntax to unwrap the underlying directory structure.
// Read more on https://github.com/hashicorp/go-getter#subdirectories
var kcBinaryDownloadLinks = map[string]string{
	"darwin/arm64":  "https://get.helm.sh/helm-v3.6.3-darwin-arm64.tar.gz//darwin-arm64",
	"linux/arm":     "https://get.helm.sh/helm-v3.6.3-linux-arm.tar.gz//linux-arm",
	"linux/ppc64le": "https://get.helm.sh/helm-v3.6.3-linux-ppc64le.tar.gz//linux-ppc64le",
	"linux/s390x":   "https://get.helm.sh/helm-v3.6.3-linux-s390x.tar.gz//linux-s390x",
	"windows/amd64": "https://get.helm.sh/helm-v3.6.3-windows-amd64.zip//windows-amd64",

	"darwin/amd64": "https://dl.k8s.io/release/v1.26.0/bin/darwin/amd64/kubectl",
	"linux/amd64":  "https://dl.k8s.io/release/v1.26.0/bin/linux/amd64/kubectl",
	"linux/arm64":  "https://dl.k8s.io/release/v1.26.0/bin/linux/arm64/kubectl",
	"linux/386":    "https://dl.k8s.io/release/v1.26.0/bin/linux/386/kubectl",
}

var _ executor.Executor = &Executor{}

// Executor provides functionality for running Helm CLI.
type Executor struct {
	pluginVersion          string
	executeCommandWithEnvs func(ctx context.Context, rawCmd string, envs map[string]string) (string, error)
}

// NewExecutor returns a new Executor instance.
func NewExecutor(ver string) *Executor {
	return &Executor{
		pluginVersion:          ver,
		executeCommandWithEnvs: pluginx.ExecuteCommandWithEnvs,
	}
}

// Metadata returns details about Helm plugin.
func (e *Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     e.pluginVersion,
		Description: description,
		//JSONSchema:  jsonSchema(),
		Dependencies: map[string]api.Dependency{
			kcBinaryName: {
				URLs: kcBinaryDownloadLinks,
			},
		},
	}, nil
}

// Execute returns a given command as response.
//
// Supported commands:
func (e *Executor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	cfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	cmd, err := normalizeCommand(in.Command)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	if err := detectNotSupportedCommands(cmd); err != nil {
		return executor.ExecuteOutput{}, err
	}
	if err := detectNotSupportedGlobalFlags(cmd); err != nil {
		return executor.ExecuteOutput{}, err
	}

	executionNs, err := e.getCommandNamespace(args)
	if err != nil {
		return "", fmt.Errorf("while extracting Namespace from command: %w", err)
	}
	if executionNs == "" { // namespace not found in command, so find default and add `-n` flag to args
		executionNs = e.findDefaultNamespace(bindings)
		args = e.addNamespaceFlag(args, executionNs)
	}

	finalArgs := e.getFinalArgs(args)
	out, err := e.cmdRunner.RunCombinedOutput(KubectlBinary, finalArgs)

	return e.handleKubectlCommand(ctx, in.Command, cfg.DefaultNamespace)
}

// Help returns help message
func (*Executor) Help(_ context.Context) (interactive.Message, error) {
	return interactive.Message{
		Base: interactive.Base{
			Body: interactive.Body{
				CodeBlock: "help()",
			},
		},
	}, nil
}

// handleHelmList construct a Kubectl CLI command and run it.
func (e *Executor) handleKubectlCommand(ctx context.Context, rawCmd string, namespace string) (executor.ExecuteOutput, error) {
	envs := map[string]string{
		"KUBECONFIG": os.Getenv("KUBECONFIG"),
	}

	out, err := e.executeCommandWithEnvs(ctx, rawCmd, envs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("%s\n%s", out, err.Error())
	}

	return executor.ExecuteOutput{
		Data: color.ClearCode(out),
	}, nil
}
