package helm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func runHelmCLIBinary(ctx context.Context, cfg Config, args []string) (string, error) {
	// Use full path if the PLUGIN_DEPENDENCY_DIR env variable is found.
	// If not, Go will get $PATH for current process and try to look up the binary.
	//
	// Unfortunately, we cannot override PATH env variable for a go-plugin process as Hashicorp overrides envs during plugin startup.
	commandName := helmBinaryName
	depDir, found := os.LookupEnv("PLUGIN_DEPENDENCY_DIR")
	if found {
		commandName = fmt.Sprintf("%s/%s", depDir, helmBinaryName)
	}

	cmd := exec.CommandContext(ctx, commandName, args...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("HELM_DRIVER=%s", cfg.HelmDriver),
		fmt.Sprintf("HELM_CACHE_HOME=%s", cfg.HelmCacheDir),
		fmt.Sprintf("HELM_CONFIG_HOME=%s", cfg.HelmConfigDir),
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}
