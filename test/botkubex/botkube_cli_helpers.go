package botkubex

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

type InstallParams struct {
	BinaryPath                              string
	HelmRepoDirectory                       string
	ConfigProviderEndpoint                  string
	ConfigProviderIdentifier                string
	ConfigProviderAPIKey                    string
	ImageRegistry                           string
	ImageRepository                         string
	ImageTag                                string
	PluginRestartPolicyThreshold            int
	PluginRestartHealthCheckIntervalSeconds int
}

func Install(t *testing.T, params InstallParams) error {
	//nolint:gosec // this is not production code
	cmd := exec.Command(params.BinaryPath, "install",
		"--auto-approve",
		"--verbose",
		fmt.Sprintf("--repo=%s", params.HelmRepoDirectory),
		fmt.Sprintf("--values=%s/botkube/e2e-cloud-test-values.yaml", params.HelmRepoDirectory),
		`--version=""`, // installer doesn't call Helm repo`index.yaml` when version is empty, so local Helm chart works as expected.
		"--set",
		fmt.Sprintf("image.registry=%s", params.ImageRegistry),
		"--set",
		fmt.Sprintf("image.repository=%s", params.ImageRepository),
		"--set",
		fmt.Sprintf("image.tag=%s", params.ImageTag),
		"--set",
		fmt.Sprintf("config.provider.endpoint=%s", params.ConfigProviderEndpoint),
		"--set",
		fmt.Sprintf("config.provider.identifier=%s", params.ConfigProviderIdentifier),
		"--set",
		fmt.Sprintf("extraEnv[0].name=%s", "BOTKUBE_PLUGINS_RESTART__POLICY_THRESHOLD"),
		"--set-string",
		fmt.Sprintf("extraEnv[0].value=%d", params.PluginRestartPolicyThreshold),
		"--set",
		fmt.Sprintf("extraEnv[1].name=%s", "BOTKUBE_PLUGINS_HEALTH__CHECK__INTERVAL"),
		"--set-string",
		fmt.Sprintf("extraEnv[1].value=%ds", params.PluginRestartHealthCheckIntervalSeconds),
		"--set",
		fmt.Sprintf("extraEnv[2].name=%s", "BOTKUBE_SETTINGS_UPGRADE_NOTIFIER"),
		"--set-string",
		"extraEnv[2].value=false",
		"--set",
		fmt.Sprintf("config.provider.apiKey=%s", params.ConfigProviderAPIKey))
	t.Logf("Executing command: %s", cmd.String())
	cmd.Env = os.Environ()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Uninstall(t *testing.T, binaryPath string) {
	//nolint:gosec // this is not production code
	cmd := exec.Command(binaryPath, "uninstall", "--release-name", "botkube", "--namespace", "botkube", "--auto-approve")
	cmd.Env = os.Environ()

	o, err := cmd.CombinedOutput()
	t.Logf("CLI output:\n%s", string(o))
	require.NoError(t, err)
}
