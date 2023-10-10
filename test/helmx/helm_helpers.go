package helmx

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// InstallChartParams are parameters for InstallChart.
type InstallChartParams struct {
	RepoName  string
	RepoURL   string
	Name      string
	Namespace string
	Command   string
}

type versionsResponse struct {
	Version string `json:"version"`
}

// ToOptions converts Command to helm install options.
func (p *InstallChartParams) ToOptions() []string {
	cmd := strings.Replace(p.Command, "\n", "", -1)
	cmd = strings.Replace(cmd, "\\", " ", -1)
	return strings.Fields(cmd)[1:]
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

	t.Logf("Installing chart %s with command %s", params.Name, params.ToOptions())
	//nolint:gosec // this is not production code
	cmd = exec.Command("helm", params.ToOptions()...)
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
