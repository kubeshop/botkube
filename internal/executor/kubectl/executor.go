package kubectl

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/executor/kubectl/accessreview"
	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

const (
	// PluginName is the name of the Helm Botkube plugin.
	PluginName       = "kubectl"
	defaultNamespace = "default"
	description      = "Run the Kubectl CLI commands directly from your favorite communication platform."
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
}

// NewExecutor returns a new Executor instance.
func NewExecutor(ver string, kcRunner kcRunner) *Executor {
	return &Executor{
		pluginVersion: ver,
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

	if err := cfg.Validate(); err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while validating configuration: %w", err)
	}

	log := loggerx.New(cfg.Log)

	cmd, err := normalizeCommand(in.Command)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	if builder.ShouldHandle(cmd) {
		guard, k8sCli, err := getBuilderDependencies(log, os.Getenv("KUBECONFIG")) // TODO: take kubeconfig from execution context
		if err != nil {
			return executor.ExecuteOutput{}, fmt.Errorf("while creating builder dependecies: %w", err)
		}

		kcBuilder := builder.NewKubectl(e.kcRunner, cfg.InteractiveBuilder, log, guard, cfg.DefaultNamespace, k8sCli.CoreV1().Namespaces(), accessreview.NewK8sAuth(k8sCli.AuthorizationV1()))
		msg, err := kcBuilder.Handle(ctx, cmd, in.Context.IsInteractivitySupported, in.Context.SlackState)
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

// Help returns help message.
func (*Executor) Help(context.Context) (api.Message, error) {
	return api.NewCodeBlockMessage(help(), true), nil
}

func getBuilderDependencies(log logrus.FieldLogger, kubeconfig string) (*command.CommandGuard, *kubernetes.Clientset, error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating kube config: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating discovery client: %w", err)
	}
	guard := command.NewCommandGuard(log, discoveryClient)
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating typed k8s client: %w", err)
	}

	return guard, k8sCli, nil
}
