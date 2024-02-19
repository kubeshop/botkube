package kubectl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/plugin"
)

const binaryName = "kubectl"

// BinaryRunner runs a kubectl binary.
type BinaryRunner struct {
	executeCommand func(ctx context.Context, rawCmd string, mutators ...plugin.ExecuteCommandMutation) (plugin.ExecuteCommandOutput, error)
}

// NewBinaryRunner returns a new BinaryRunner instance.
func NewBinaryRunner() *BinaryRunner {
	return &BinaryRunner{
		executeCommand: plugin.ExecuteCommand,
	}
}

// RunKubectlCommand runs a Kubectl CLI command and run output.
func (e *BinaryRunner) RunKubectlCommand(ctx context.Context, kubeConfigPath, defaultNamespace, cmd string) (string, error) {
	if err := detectNotSupportedCommands(cmd); err != nil {
		return "", err
	}
	if err := detectNotSupportedGlobalFlags(cmd); err != nil {
		return "", err
	}

	if strings.EqualFold(cmd, "options") {
		return optionsCommandOutput(), nil
	}

	isNs, err := isNamespaceFlagSet(cmd)
	if err != nil {
		return "", err
	}

	if !isNs {
		// appending the defaultNamespace at the beginning to do not break the command e.g.
		//    kubectl exec mypod -- date
		cmd = fmt.Sprintf("-n %s %s", defaultNamespace, cmd)
	}

	envs := map[string]string{
		"KUBECONFIG": kubeConfigPath,
	}

	runCmd := fmt.Sprintf("%s %s", binaryName, cmd)
	out, err := e.executeCommand(ctx, runCmd, plugin.ExecuteCommandEnvs(envs))
	if err != nil {
		return "", err
	}

	return color.ClearCode(out.Stdout), nil
}

// getAllNamespaceFlag returns the namespace value extracted from a given args.
// If `--A, --all-namespaces` or `--namespace/-n` was found, returns true.
func isNamespaceFlagSet(cmd string) (bool, error) {
	f := pflag.NewFlagSet("extract-ns", pflag.ContinueOnError)
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags as there are specially process by pflag")

	// ignore unknown flags errors, e.g. `--cluster-name` etc.
	f.ParseErrorsWhitelist.UnknownFlags = true

	var isNs string
	f.StringVarP(&isNs, "namespace", "n", "", "Kubernetes Namespace")

	var isAllNs bool
	f.BoolVarP(&isAllNs, "all-namespaces", "A", false, "Kubernetes All Namespaces")
	if err := f.Parse(strings.Fields(cmd)); err != nil {
		return false, err
	}
	return isAllNs || isNs != "", nil
}

// KubeconfigScopedRunner is a runner that executes kubectl commands using a specific kubeconfig file.
type KubeconfigScopedRunner struct {
	underlying     kcRunner
	kubeconfigPath string
}

// NewKubeconfigScopedRunner creates a new instance of KubeconfigScopedRunner.
func NewKubeconfigScopedRunner(underlying kcRunner, kubeconfigPath string) *KubeconfigScopedRunner {
	return &KubeconfigScopedRunner{underlying: underlying, kubeconfigPath: kubeconfigPath}
}

// RunKubectlCommand runs a kubectl CLI command scoped to configured kubeconfig.
func (k *KubeconfigScopedRunner) RunKubectlCommand(ctx context.Context, defaultNamespace, cmd string) (string, error) {
	if k.kubeconfigPath == "" {
		return "", errors.New("kubeconfig is missing")
	}

	return k.underlying.RunKubectlCommand(ctx, k.kubeconfigPath, defaultNamespace, cmd)
}
