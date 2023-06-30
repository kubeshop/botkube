package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	"golang.org/x/oauth2"

	"github.com/kubeshop/botkube/test/discordx"
	"github.com/kubeshop/botkube/test/helmx"
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

	token, err := refreshAccessToken(t, appCfg)
	require.NoError(t, err)

	t.Log("Pruning old instances...")
	err = pruneInstances(t, appCfg.BotkubeCloudDevGQLEndpoint, token)
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

func pruneInstances(t *testing.T, url, token string) error {
	t.Helper()

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := graphql.NewClient(url, httpClient)

	var query struct {
		Deployments struct {
			Data []struct {
				ID string `graphql:"id"`
			}
		} `graphql:"deployments()"`
	}
	err := client.Query(context.Background(), &query, map[string]interface{}{})
	require.NoError(t, err)

	for _, deployment := range query.Deployments.Data {
		var mutation struct {
			Success bool `graphql:"deleteDeployment(id: $id)"`
		}
		err := client.Mutate(context.Background(), &mutation, map[string]interface{}{
			"id": graphql.ID(deployment.ID),
		})
		require.NoError(t, err)
	}
	return nil
}

func refreshAccessToken(t *testing.T, cfg Config) (string, error) {
	t.Helper()

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
