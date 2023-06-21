package helmx

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type InstallChartParams struct {
	RepoName  string
	RepoURL   string
	Name      string
	Namespace string
	Command   string
}

func (p *InstallChartParams) ToOptions() []string {
	cmd := strings.Replace(p.Command, "\n", "", -1)
	cmd = strings.Replace(cmd, "\\", " ", -1)
	return strings.Fields(cmd)[1:]
}

func InstallChart(t *testing.T, params InstallChartParams) func(t *testing.T) {
	t.Helper()

	t.Logf("Adding helm repository %s with url %s...", params.Name, params.RepoURL)
	cmd := exec.Command("helm", "repo", "add", params.RepoName, params.RepoURL)
	repoAddOutput, err := cmd.CombinedOutput()
	t.Log(string(repoAddOutput))
	require.NoError(t, err)

	t.Log("Updating repo...")
	cmd = exec.Command("helm", "repo", "update", params.RepoName)
	repoUpdateOutput, err := cmd.CombinedOutput()
	t.Log(string(repoUpdateOutput))
	require.NoError(t, err)

	t.Logf("Installing chart %s with command %s", params.Name, params.ToOptions())
	cmd = exec.Command("helm", params.ToOptions()...)
	installOutput, err := cmd.CombinedOutput()
	t.Log(string(installOutput))
	require.NoError(t, err)

	return func(t *testing.T) {
		t.Helper()

		cmd := exec.Command("helm", "del", params.Name, "-n", params.Namespace)
		delOutput, err := cmd.CombinedOutput()
		t.Log(string(delOutput))
		require.NoError(t, err)
	}
}
