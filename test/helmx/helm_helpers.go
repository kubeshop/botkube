package helmx

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/stretchr/testify/require"
)

var apiKeyRegex = regexp.MustCompile(`(.*)=\s*key:[a-fA-F0-9-]+`)

// InstallChartParams are parameters for InstallChart.
type InstallChartParams struct {
	RepoName      string
	RepoURL       string
	Name          string
	Namespace     string
	Command       string
	PluginRepoURL string
}

type versionsResponse struct {
	Version string `json:"version"`
}

// ToOptions converts Command to helm install options.
func (p *InstallChartParams) ToOptions(version string) []string {
	cmd := strings.Replace(p.Command, "\n", "", -1)
	cmd = strings.Replace(cmd, "\\", " ", -1)
	versionRegex := regexp.MustCompile(`--version (\S+)`)
	cmd = versionRegex.ReplaceAllString(cmd, "--version "+version)
	cmdParts := strings.Fields(cmd)[1:]
	if p.PluginRepoURL == "" {
		return cmdParts
	}
	extraEnvs := []string{
		"--set",
		fmt.Sprintf("extraEnv[0].name=%s", "BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL"),
		"--set-string",
		fmt.Sprintf("extraEnv[0].value=%s", p.PluginRepoURL),
	}
	cmdParts = append(cmdParts, extraEnvs...)
	return cmdParts
}

// InstallChart installs helm chart.
func InstallChart(t *testing.T, params InstallChartParams) func(t *testing.T) {
	t.Helper()

	t.Logf("Adding helm repository %s with url %s...", params.Name, params.RepoURL)
	//nolint:gosec // this is not production code
	cmd := exec.Command("helm", "repo", "add", params.RepoName, params.RepoURL)
	repoAddOutput, err := cmd.CombinedOutput()
	t.Log(string(repoAddOutput))
	require.NoError(t, err)

	t.Log("Updating repo...")
	//nolint:gosec // this is not production code
	cmd = exec.Command("helm", "repo", "update", params.RepoName)
	repoUpdateOutput, err := cmd.CombinedOutput()
	t.Log(string(repoUpdateOutput))
	require.NoError(t, err)

	t.Log("Finding latest version...")
	cmd = exec.Command("helm", "search", "repo", params.RepoName, "--devel", "--versions", "-o", "json") // #nosec G204
	versionsOutput, err := cmd.CombinedOutput()
	require.NoError(t, err)
	latestVersion := latestVersion(t, versionsOutput)
	t.Logf("Found version: %s", latestVersion)

	helmOpts := params.ToOptions(latestVersion)
	t.Logf("Installing chart %s with parameters %+v", params.Name, redactAPIKey(helmOpts))
	//nolint:gosec // this is not production code
	cmd = exec.Command("helm", helmOpts...)
	installOutput, err := cmd.CombinedOutput()
	t.Log(string(installOutput))
	require.NoError(t, err)

	return func(t *testing.T) {
		t.Helper()

		//nolint:gosec // this is not production code
		cmd := exec.Command("helm", "del", params.Name, "-n", params.Namespace)
		delOutput, err := cmd.CombinedOutput()
		t.Log(string(delOutput))
		require.NoError(t, err)
	}
}

func latestVersion(t *testing.T, versionsOutput []byte) string {
	var versions []versionsResponse
	err := json.Unmarshal(versionsOutput, &versions)
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	return versions[0].Version
}

func redactAPIKey(in []string) []string {
	dst := make([]string, len(in))
	copy(dst, in)

	for i := range dst {
		dst[i] = apiKeyRegex.ReplaceAllString(dst[i], "$1=REDACTED")
	}
	return dst
}

const (
	uninstallPollInterval = 1 * time.Second
	uninstallTimeout      = 30 * time.Second
)

// WaitForUninstallation waits until a Helm chart is uninstalled, based on the atomic value.
// It's a workaround for the Helm chart uninstallation issue.
// We have a glitch on backend side and the logic below is a workaround for that.
// Tl;dr uninstalling Helm chart reports "DISCONNECTED" status, and deployment deletion reports "DELETED" status.
// If we do these two things too quickly, we'll run into resource version mismatch in repository logic.
// Read more here: https://github.com/kubeshop/botkube-cloud/pull/486#issuecomment-1604333794
func WaitForUninstallation(ctx context.Context, t *testing.T, alreadyUninstalled *atomic.Bool) error {
	t.Helper()
	t.Log("Waiting for Helm chart uninstallation, in order to proceed with deleting Botkube Cloud instance...")
	err := wait.PollUntilContextTimeout(ctx, uninstallPollInterval, uninstallTimeout, false, func(ctx context.Context) (done bool, err error) {
		return alreadyUninstalled.Load(), nil
	})
	waitInterrupted := wait.Interrupted(err)
	if err != nil && !waitInterrupted {
		return err
	}

	if waitInterrupted {
		t.Log("Waiting for Helm chart uninstallation timed out. Proceeding with deleting other resources...")
		return nil
	}

	t.Log("Waiting a bit more...")
	time.Sleep(3 * time.Second) // ugly, but at least we will be pretty sure we won't run into the resource version mismatch
	return nil
}
