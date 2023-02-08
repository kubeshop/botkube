package kubectl

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

const (
	// PluginName is the name of the Helm Botkube plugin.
	PluginName       = "kubectl"
	defaultNamespace = "default"
	description      = "Kubectl is the Botkube executor plugin that allows you to run the Kubectl CLI commands directly from any communication platform."
)

var kcBinaryDownloadLinks = map[string]string{
	"windows/amd64": "https://dl.k8s.io/release/v1.26.0/bin/windows/amd64/kubectl.exe",
	"darwin/amd64":  "https://dl.k8s.io/release/v1.26.0/bin/darwin/amd64/kubectl",
	"darwin/arm64":  "https://dl.k8s.io/release/v1.26.0/bin/darwin/arm64/kubectl",
	"linux/amd64":   "https://dl.k8s.io/release/v1.26.0/bin/linux/amd64/kubectl",
	"linux/s390x":   "https://dl.k8s.io/release/v1.26.0/bin/linux/s390x/kubectl",
	"linux/ppc64le": "https://dl.k8s.io/release/v1.26.0/bin/linux/ppc64le/kubectl",
	"linux/arm64":   "https://dl.k8s.io/release/v1.26.0/bin/linux/arm64/kubectl",
	"linux/386":     "https://dl.k8s.io/release/v1.26.0/bin/linux/386/kubectl",
}

var _ executor.Executor = &Executor{}

type (
	kcRunner interface {
		RunKubectlCommand(ctx context.Context, defaultNamespace, cmd string) (string, error)
	}
)

// Executor provides functionality for running Helm CLI.
type Executor struct {
	pluginVersion string
	kcRunner      kcRunner

	logger    logrus.FieldLogger
	kcBuilder *builder.KubectlCmdBuilder
}

// NewExecutor returns a new Executor instance.
func NewExecutor(logger logrus.FieldLogger, ver string, kcRunner kcRunner) *Executor {
	return &Executor{
		pluginVersion: ver,
		logger:        logger,
		kcBuilder:     builder.NewKubectlCmdBuilder(),
		kcRunner:      kcRunner,
	}
}

// Metadata returns details about Helm plugin.
func (e *Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     e.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(description),
		Dependencies: map[string]api.Dependency{
			binaryName: {
				URLs: kcBinaryDownloadLinks,
			},
		},
	}, nil
}

// Execute returns a given command as response.
func (e *Executor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	cfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	cmd, err := normalizeCommand(in.Command)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	if e.kcBuilder.ShouldHandle(cmd) {
		msg, err := e.kcBuilder.Handle(ctx, e.logger, in.Context.IsInteractivitySupported)
		if err != nil {
			return executor.ExecuteOutput{}, fmt.Errorf("while running command builder: %w", err)
		}
		return executor.ExecuteOutput{
			Message: msg,
		}, nil
	}

	out, err := e.kcRunner.RunKubectlCommand(ctx, cfg.DefaultNamespace, cmd)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	return executor.ExecuteOutput{
		Message: api.NewCodeBlockMessage(out, true),
	}, nil
}

// Help returns help message
func (*Executor) Help(_ context.Context) (api.Message, error) {
	return api.NewCodeBlockMessage(help(), true), nil
}
