package helm

import (
	"context"
	"fmt"
	"os/exec"
)

func runHelmCLIBinary(ctx context.Context, cfg Config, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, helmBinaryName, args...)
	cmd.Env = []string{
		fmt.Sprintf("HELM_DRIVER=%s", cfg.HelmDriver),
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
