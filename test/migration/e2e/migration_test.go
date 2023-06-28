package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"

	"github.com/kubeshop/botkube/test/migration/discordx"
	"github.com/kubeshop/botkube/test/migration/helmx"
)

const (
	helmCmd = `helm upgrade botkube --install --version v1.1.0 --namespace botkube --create-namespace --wait \
	--set communications.default-group.discord.enabled=true \
	--set communications.default-group.discord.channels.default.id=%s \
	--set communications.default-group.discord.botID=%s \
	--set communications.default-group.discord.token=%s \
	--set settings.clusterName=%s \
	--set executors.k8s-default-tools.botkube/kubectl.enabled=true \
	--set analytics.disable=true \
	botkube/botkube`
)

type Config struct {
	APIToken    string
	GQLEndpoint string
	Discord     discordx.DiscordConfig
}

func TestBotkubeMigration(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	t.Log("Initializing Discord...")
	tester, err := discordx.New(appCfg.Discord)
	require.NoError(t, err)

	t.Log("Initializing users...")
	tester.InitUsers(t)

	t.Log("Creating channel...")
	channel, createChannelCallback := tester.CreateChannel(t, "test-migration")

	t.Cleanup(func() { createChannelCallback(t) })

	t.Logf("Channel %s", channel.Name)

	cmd := fmt.Sprintf(helmCmd, channel.ID, appCfg.Discord.BotID, appCfg.Discord.BotToken, "TestMigration")
	params := helmx.InstallChartParams{
		RepoURL:   "https://charts.botkube.io",
		RepoName:  "botkube",
		Name:      "botkube",
		Namespace: "botkube",
		Command:   cmd,
	}
	helmInstallCallback := helmx.InstallChart(t, params)
	t.Cleanup(func() { helmInstallCallback(t) })

	t.Run("Migrate Discord Botkube to Botkube Cloud", func(t *testing.T) {
		fmt.Println("Endpoint", appCfg.GQLEndpoint)
		fmt.Println("Using tk for cloud: ", appCfg.APIToken)
		cmd := exec.Command(os.Getenv("BOTKUBE_BIN"), "migrate",
			fmt.Sprintf("--token=%s", appCfg.APIToken),
			fmt.Sprintf("--cloud-api-url=%s", appCfg.GQLEndpoint),
			"--instance-name=test-migration",
			"-q")
		cmd.Env = os.Environ()

		o, err := cmd.CombinedOutput()
		require.NoError(t, err, string(o))
	})

}
