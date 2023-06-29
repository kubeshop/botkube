package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
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
	oauthURL = "https://botkube-dev.eu.auth0.com/oauth/token"
)

type Config struct {
	BotkubeCloudDevGQLEndpoint   string
	BotkubeCloudDevRefreshToken  string
	BotkubeCloudDevAuth0ClientID string
	Discord                      discordx.DiscordConfig
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
		token, err := refreshAccessToken(appCfg)
		require.NoError(t, err)
		cmd := exec.Command(os.Getenv("BOTKUBE_BIN"), "migrate",
			fmt.Sprintf("--token=%s", token),
			fmt.Sprintf("--cloud-api-url=%s", appCfg.BotkubeCloudDevGQLEndpoint),
			"--instance-name=test-migration",
			"-q")
		cmd.Env = os.Environ()

		o, err := cmd.CombinedOutput()
		require.NoError(t, err, string(o))
	})

}

func refreshAccessToken(cfg Config) (string, error) {
	type TokenMsg struct {
		Token string `json:"access_token"`
	}

	payloadRequest := fmt.Sprintf("grant_type=refresh_token&client_id=%s&refresh_token=%s", cfg.BotkubeCloudDevAuth0ClientID, cfg.BotkubeCloudDevRefreshToken)
	payload := strings.NewReader(payloadRequest)
	req, err := http.NewRequest("POST", oauthURL, payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	var client = &http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to get response")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	var tokenMsg TokenMsg
	if err := json.Unmarshal(body, &tokenMsg); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response body")
	}
	return tokenMsg.Token, nil
}
