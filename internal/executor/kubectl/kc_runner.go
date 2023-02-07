package kubectl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/color"

	"github.com/kubeshop/botkube/pkg/pluginx"
)

const binaryName = "kubectl"

type BinaryRunner struct {
	executeCommandWithEnvs func(ctx context.Context, rawCmd string, envs map[string]string) (string, error)
}

func NewBinaryRunner() *BinaryRunner {
	return &BinaryRunner{
		executeCommandWithEnvs: pluginx.ExecuteCommandWithEnvs,
	}
}

// RunKubectlCommand runs a Kubectl CLI command and run output.
func (e *BinaryRunner) RunKubectlCommand(ctx context.Context, defaultNamespace, cmd string) (string, error) {
	if err := detectNotSupportedCommands(cmd); err != nil {
		return "", err
	}
	if err := detectNotSupportedGlobalFlags(cmd); err != nil {
		return "", err
	}

	if strings.EqualFold(cmd, "options") {
		return optionsCommandOutput(), nil
	}

	if !isNamespaceFlagSet(cmd) {
		// appending the defaultNamespace at the beginning to do not break the command e.g.
		//    kubectl exec mypod -- date
		cmd = fmt.Sprintf("-n %s %s", defaultNamespace, cmd)
	}

	envs := map[string]string{
		// TODO: take it from the execute context.
		"KUBECONFIG": os.Getenv("KUBECONFIG"),
	}

	runCmd := fmt.Sprintf("%s %s", binaryName, cmd)
	out, err := e.executeCommandWithEnvs(ctx, runCmd, envs)
	if err != nil {
		return "", fmt.Errorf("%s\n%s", out, err.Error())
	}

	return color.ClearCode(out), nil
}

// isNamespaceFlagSet returns true if `--namespace/-n` was found.
func isNamespaceFlagSet(cmd string) bool {
	return strings.Contains(cmd, "-n") || strings.Contains(cmd, "--namespace")
}
