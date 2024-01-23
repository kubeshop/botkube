//go:build migration

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	"golang.org/x/oauth2"

	"github.com/kubeshop/botkube/internal/ptr"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
	"github.com/kubeshop/botkube/test/commplatform"
	"github.com/kubeshop/botkube/test/helmx"
)

const (
	// using latest version (without `--version` flag)
	helmCmdFmt = `helm upgrade botkube --install --namespace botkube --create-namespace --wait \
	--set communications.default-group.discord.enabled=true \
	--set communications.default-group.discord.channels.default.id=%s \
	--set communications.default-group.discord.botID=%s \
	--set communications.default-group.discord.token=%s \
	--set settings.clusterName=%s \
	--set executors.k8s-default-tools.botkube/kubectl.enabled=true \
	--set executors.k8s-default-tools.botkube/helm.enabled=true \
	--set analytics.disable=true \
	--set image.tag=v9.99.9-dev \
	--set plugins.repositories.botkube.url=https://storage.googleapis.com/botkube-plugins-latest/plugins-index.yaml \
	botkube/botkube`

	//nolint:gosec // false positive
	oauth2TokenURL = "https://botkube-dev.eu.auth0.com/oauth/token"
)

var (
	defaultRBAC = &gqlModel.Rbac{
		User: &gqlModel.UserPolicySubject{
			Type:   gqlModel.PolicySubjectTypeEmpty,
			Static: &gqlModel.UserStaticSubject{},
			Prefix: nil,
		},
		Group: &gqlModel.GroupPolicySubject{
			Type: gqlModel.PolicySubjectTypeStatic,
			Static: &gqlModel.GroupStaticSubject{
				Values: []string{"botkube-plugins-default"},
			},
			Prefix: ptr.FromType(""),
		},
	}
)

type MigrationConfig struct {
	BotkubeCloudDevGQLEndpoint   string
	BotkubeCloudDevRefreshToken  string
	BotkubeCloudDevAuth0ClientID string
	BotkubeBinaryPath            string
	DeploymentName               string `envconfig:"default=test-migration"`
	Discord                      commplatform.DiscordConfig
	DiscordBotToken              string

	Timeout time.Duration `envconfig:"default=15s"`
}

func TestBotkubeMigration(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg MigrationConfig
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)
	appCfg.Discord.RecentMessagesLimit = 2 // fix the value for this specific test

	token, err := refreshAccessToken(t, appCfg)
	require.NoError(t, err)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	gqlCli := graphql.NewClient(appCfg.BotkubeCloudDevGQLEndpoint, httpClient)

	t.Log("Pruning old instances...")
	pruneInstances(t, gqlCli)

	t.Log("Initializing Discord...")
	tester, err := commplatform.NewDiscordTester(appCfg.Discord)
	require.NoError(t, err)

	t.Log("Initializing users...")
	tester.InitUsers(t)

	t.Log("Creating channel...")
	channel, createChannelCallback := tester.CreateChannel(t, "migration")
	t.Cleanup(func() { createChannelCallback(t) })

	t.Log("Inviting Bot to the channel...")
	tester.InviteBotToChannel(t, channel.ID())

	t.Logf("Channel %s", channel.Name())

	cmd := fmt.Sprintf(helmCmdFmt, channel.ID(), appCfg.Discord.BotID, appCfg.DiscordBotToken, channel.Name())
	params := helmx.InstallChartParams{
		RepoName:  "botkube",
		RepoURL:   "https://charts.botkube.io",
		Name:      "botkube",
		Namespace: "botkube",
		Command:   cmd,
	}
	helmInstallCallback := helmx.InstallChart(t, params) // TODO: Fix - do not install static Botkube version
	t.Cleanup(func() { helmInstallCallback(t) })

	t.Run("Check if Botkube is running before migration", func(t *testing.T) {
		clusterName := channel.Name()

		// Discord bot needs a bit more time to connect to Discord API.
		time.Sleep(appCfg.Discord.MessageWaitTimeout)

		t.Log("Waiting for Bot message in channel...")
		err = tester.WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", clusterName))
		require.NoError(t, err)

		t.Log("Testing ping...")
		command := "ping"
		expectedMessage := fmt.Sprintf("`%s` on `%s`\n```\npong", command, clusterName)
		tester.PostMessageToBot(t, channel.ID(), command)
		err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
		require.NoError(t, err)
	})

	t.Run("Migrate Discord Botkube to Botkube Cloud", func(t *testing.T) {
		//nolint:gosec // this is not production code
		cmd := exec.Command(appCfg.BotkubeBinaryPath, "migrate",
			"--auto-approve",
			"--skip-open-browser",
			"--verbose",
			"--image-tag=v9.99.9-dev",
			fmt.Sprintf("--token=%s", token),
			fmt.Sprintf("--cloud-api-url=%s", appCfg.BotkubeCloudDevGQLEndpoint),
			fmt.Sprintf("--instance-name=%s", appCfg.DeploymentName))
		cmd.Env = os.Environ()

		o, err := cmd.CombinedOutput()
		t.Logf("CLI output:\n%s", string(o))
		require.NoError(t, err)
	})

	t.Run("Check if the instance is created on Botkube Cloud side", func(t *testing.T) {
		deployPage := queryInstances(t, gqlCli)
		require.Len(t, deployPage.Data, 1)

		deploy := deployPage.Data[0]
		assert.Equal(t, appCfg.DeploymentName, deploy.Name)

		assertAliases(t, deploy.Aliases)
		assertPlatforms(t, deploy.Platforms, appCfg, channel.ID())
		assertPlugins(t, deploy.Plugins)
	})

	t.Run("Check if Botkube Cloud is running after migration", func(t *testing.T) {
		// Discord bot needs a bit more time to connect to Discord API.
		time.Sleep(appCfg.Discord.MessageWaitTimeout)

		clusterName := appCfg.DeploymentName // it is different after migration

		t.Log("Waiting for Bot message in channel...")
		err = tester.WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", clusterName))
		assert.NoError(t, err)

		t.Log("Testing ping...")
		command := "ping"
		expectedMessage := fmt.Sprintf("`%s` on `%s`\n```\npong", command, clusterName)
		tester.PostMessageToBot(t, channel.ID(), command)
		err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
		assert.NoError(t, err)
	})
}

func queryInstances(t *testing.T, client *graphql.Client) gqlModel.DeploymentPage {
	t.Helper()
	var query struct {
		Deployments gqlModel.DeploymentPage `graphql:"deployments"`
	}
	err := client.Query(context.Background(), &query, nil)
	require.NoError(t, err)

	return query.Deployments
}

func pruneInstances(t *testing.T, client *graphql.Client) {
	t.Helper()

	deployPage := queryInstances(t, client)
	for _, deployment := range deployPage.Data {
		var mutation struct {
			Success bool `graphql:"deleteDeployment(id: $id)"`
		}
		err := client.Mutate(context.Background(), &mutation, map[string]interface{}{
			"id": graphql.ID(deployment.ID),
		})
		require.NoError(t, err)
	}
}

func refreshAccessToken(t *testing.T, cfg MigrationConfig) (string, error) {
	t.Helper()

	type TokenMsg struct {
		Token string `json:"access_token"`
	}

	payloadRequest := fmt.Sprintf("grant_type=refresh_token&client_id=%s&refresh_token=%s", cfg.BotkubeCloudDevAuth0ClientID, cfg.BotkubeCloudDevRefreshToken)
	payload := strings.NewReader(payloadRequest)
	req, err := http.NewRequest("POST", oauth2TokenURL, payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	var client = &http.Client{
		Timeout: cfg.Timeout,
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

func assertAliases(t *testing.T, actual []*gqlModel.Alias) {
	t.Helper()

	expected := []*gqlModel.Alias{
		{
			ID:          "",
			Name:        "k",
			DisplayName: "Kubectl alias",
			Command:     "kubectl",
			Deployments: nil,
		},
		{
			ID:          "",
			Name:        "kc",
			DisplayName: "Kubectl alias",
			Command:     "kubectl",
			Deployments: nil,
		},
		{
			ID:          "",
			Name:        "x",
			DisplayName: "Exec alias",
			Command:     "exec",
			Deployments: nil,
		},
		{
			ID:          "",
			Name:        "chatgpt",
			DisplayName: "Doctor alias",
			Command:     "doctor",
			Deployments: nil,
		},
	}

	assert.Len(t, actual, 4)

	// trim ID and deployments
	for i := range actual {
		actual[i].ID = ""
		actual[i].Deployments = nil
	}

	assert.ElementsMatchf(t, expected, actual, "Aliases are not equal")
}

func assertPlatforms(t *testing.T, actual *gqlModel.Platforms, appCfg MigrationConfig, channelID string) {
	t.Helper()

	expectedDiscords := []*gqlModel.Discord{
		{
			ID:    "", // trim
			Name:  "", // trim
			Token: appCfg.DiscordBotToken,
			BotID: appCfg.Discord.BotID,
			Channels: []*gqlModel.ChannelBindingsByID{
				{
					ID: channelID,
					Bindings: &gqlModel.BotBindings{
						Sources: []string{
							"k8s-err-events",
							"k8s-recommendation-events",
							"k8s-err-events-with-ai-support",
							"argocd",
						},
						Executors: []string{
							"k8s-default-tools",
							"bins-management",
							"ai",
							"flux",
						},
					},
					NotificationsDisabled: ptr.FromType(false),
				},
			},
		},
	}

	assert.NotNil(t, actual, 1)
	assert.Len(t, actual.Discords, 1)

	// trim ignored fields
	for i := range actual.Discords {
		actual.Discords[i].ID = ""
		actual.Discords[i].Name = ""
		trimAutoGeneratedSuffixes(actual.Discords[i].Channels)
	}

	assert.ElementsMatchf(t, expectedDiscords, actual.Discords, "Platforms are not equal")
}

func assertPlugins(t *testing.T, actual []*gqlModel.Plugin) {
	t.Helper()

	expectedPlugins := []*gqlModel.Plugin{
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Resource Created Events",
			Type:              "SOURCE",
			ConfigurationName: "k8s-create-events",
			Configuration:     "{\"event\":{\"types\":[\"create\"]},\"namespaces\":{\"include\":[\".*\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/configmaps\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Errors for resources with logs",
			Type:              "SOURCE",
			ConfigurationName: "k8s-err-with-logs-events",
			Configuration:     "{\"event\":{\"types\":[\"error\"]},\"namespaces\":{\"include\":[\".*\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Recommendations",
			Type:              "SOURCE",
			ConfigurationName: "k8s-recommendation-events",
			Configuration:     "{\"namespaces\":{\"include\":[\".*\"]},\"recommendations\":{\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true},\"pod\":{\"labelsSet\":true,\"noLatestImageTag\":true}}}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Info",
			Type:              "SOURCE",
			ConfigurationName: "k8s-all-events",
			Configuration:     "{\"annotations\":{},\"event\":{\"message\":{\"exclude\":[],\"include\":[]},\"reason\":{\"exclude\":[],\"include\":[]},\"types\":[\"create\",\"delete\",\"error\"]},\"filters\":{\"nodeEventsChecker\":true,\"objectAnnotationChecker\":true},\"labels\":{},\"namespaces\":{\"include\":[\".*\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"event\":{\"message\":{\"exclude\":[\".*nf_conntrack_buckets.*\"]}},\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/persistentvolumes\"},{\"type\":\"v1/persistentvolumeclaims\"},{\"type\":\"v1/configmaps\"},{\"type\":\"rbac.authorization.k8s.io/v1/roles\"},{\"type\":\"rbac.authorization.k8s.io/v1/rolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterrolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterroles\"},{\"event\":{\"types\":[\"create\",\"update\",\"delete\",\"error\"]},\"type\":\"apps/v1/daemonsets\",\"updateSetting\":{\"fields\":[\"spec.template.spec.containers[*].image\",\"status.numberReady\"],\"includeDiff\":true}},{\"event\":{\"types\":[\"create\",\"update\",\"delete\",\"error\"]},\"type\":\"batch/v1/jobs\",\"updateSetting\":{\"fields\":[\"spec.template.spec.containers[*].image\",\"status.conditions[*].type\"],\"includeDiff\":true}},{\"event\":{\"types\":[\"create\",\"update\",\"delete\",\"error\"]},\"type\":\"apps/v1/deployments\",\"updateSetting\":{\"fields\":[\"spec.template.spec.containers[*].image\",\"status.availableReplicas\"],\"includeDiff\":true}},{\"event\":{\"types\":[\"create\",\"update\",\"delete\",\"error\"]},\"type\":\"apps/v1/statefulsets\",\"updateSetting\":{\"fields\":[\"spec.template.spec.containers[*].image\",\"status.readyReplicas\"],\"includeDiff\":true}}]}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Errors",
			Type:              "SOURCE",
			ConfigurationName: "k8s-err-events",
			Configuration:     "{\"event\":{\"types\":[\"error\"]},\"namespaces\":{\"include\":[\".*\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"event\":{\"message\":{\"exclude\":[\".*nf_conntrack_buckets.*\"]}},\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/persistentvolumes\"},{\"type\":\"v1/persistentvolumeclaims\"},{\"type\":\"v1/configmaps\"},{\"type\":\"rbac.authorization.k8s.io/v1/roles\"},{\"type\":\"rbac.authorization.k8s.io/v1/rolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterrolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterroles\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubectl",
			DisplayName:       "botkube/kubectl",
			Type:              "EXECUTOR",
			ConfigurationName: "k8s-default-tools",
			Configuration:     "{\"defaultNamespace\":\"default\"}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/helm",
			DisplayName:       "botkube/helm",
			Type:              "EXECUTOR",
			ConfigurationName: "k8s-default-tools",
			Configuration:     "{\"defaultNamespace\":\"default\",\"helmCacheDir\":\"/tmp/helm/.cache\",\"helmConfigDir\":\"/tmp/helm/\",\"helmDriver\":\"secret\"}",
			Enabled:           true,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/argocd",
			DisplayName:       "botkube/argocd",
			Type:              "SOURCE",
			ConfigurationName: "argocd",
			Configuration:     "{\"argoCD\":{\"notificationsConfigMap\":{\"name\":\"argocd-notifications-cm\",\"namespace\":\"argocd\"},\"uiBaseUrl\":\"http://localhost:8080\"},\"defaultSubscriptions\":{\"applications\":[{\"name\":\"guestbook\",\"namespace\":\"argocd\"}]}}",
			Enabled:           false,
			Rbac: &gqlModel.Rbac{
				User: defaultRBAC.User,
				Group: &gqlModel.GroupPolicySubject{
					Type: defaultRBAC.Group.Type,
					Static: &gqlModel.GroupStaticSubject{
						Values: []string{"argocd"},
					},
					Prefix: defaultRBAC.Group.Prefix,
				},
			},
		},
		{
			Name:              "botkube/keptn",
			DisplayName:       "botkube/keptn",
			Type:              "SOURCE",
			ConfigurationName: "keptn",
			Configuration:     "{\"log\":{\"level\":\"info\"},\"project\":\"\",\"service\":\"\",\"token\":\"\",\"url\":\"http://api-gateway-nginx.keptn.svc.cluster.local/api\"}",
			Enabled:           false,
			Rbac: &gqlModel.Rbac{
				User: defaultRBAC.User,
				Group: &gqlModel.GroupPolicySubject{
					Type: defaultRBAC.Group.Type,
					Static: &gqlModel.GroupStaticSubject{
						Values: []string{},
					},
					Prefix: nil,
				},
			},
		},
		{
			Name:              "botkube/exec",
			DisplayName:       "botkube/exec",
			Type:              "EXECUTOR",
			ConfigurationName: "bins-management",
			Configuration:     "{\"templates\":[{\"ref\":\"github.com/kubeshop/botkube//cmd/executor/exec/templates?ref=v1.7.0\"}]}",
			Enabled:           false,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/doctor",
			DisplayName:       "botkube/doctor",
			Type:              "EXECUTOR",
			ConfigurationName: "ai",
			Configuration:     "{\"apiBaseUrl\":\"\",\"apiKey\":\"\",\"defaultEngine\":\"\",\"organizationID\":\"\",\"userAgent\":\"\"}",
			Enabled:           false,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/flux",
			DisplayName:       "botkube/flux",
			Type:              "EXECUTOR",
			ConfigurationName: "flux",
			Configuration:     "{\"github\":{\"auth\":{\"accessToken\":\"\"}},\"log\":{\"level\":\"info\"}}",
			Enabled:           false,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/kubernetes",
			DisplayName:       "Kubernetes Errors with AI support",
			Type:              "SOURCE",
			ConfigurationName: "k8s-err-events-with-ai-support",
			Configuration:     "{\"event\":{\"types\":[\"error\"]},\"extraButtons\":[{\"button\":{\"commandTpl\":\"doctor --resource={{ .Kind | lower }}/{{ .Name }} --namespace={{ .Namespace }} --error={{ .Reason }} --bk-cmd-header='AI assistance'\",\"displayName\":\"Get Help\"},\"enabled\":true,\"trigger\":{\"type\":[\"error\"]}}],\"namespaces\":{\"include\":[\".*\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"event\":{\"message\":{\"exclude\":[\".*nf_conntrack_buckets.*\"]}},\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/persistentvolumes\"},{\"type\":\"v1/persistentvolumeclaims\"},{\"type\":\"v1/configmaps\"},{\"type\":\"rbac.authorization.k8s.io/v1/roles\"},{\"type\":\"rbac.authorization.k8s.io/v1/rolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterrolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterroles\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}",
			Enabled:           false,
			Rbac:              defaultRBAC,
		},
		{
			Name:              "botkube/prometheus",
			DisplayName:       "botkube/prometheus",
			Type:              "SOURCE",
			ConfigurationName: "prometheus",
			Configuration:     "{\"alertStates\":[\"firing\",\"pending\",\"inactive\"],\"ignoreOldAlerts\":true,\"log\":{\"level\":\"info\"},\"url\":\"http://localhost:9090\"}",
			Enabled:           false,
			Rbac:              defaultRBAC,
		},
	}

	assert.NotEmpty(t, actual)

	// trim ignored fields
	for i := range actual {
		actual[i].ID = ""
		actual[i].Rbac.ID = ""

		parts := strings.Split(actual[i].ConfigurationName, "_")
		actual[i].ConfigurationName = parts[0]
	}

	assert.ElementsMatchf(t, expectedPlugins, actual, "Plugins are not equal")
}

func trimAutoGeneratedSuffixes(channels []*gqlModel.ChannelBindingsByID) {
	for c := range channels {
		for i, s := range channels[c].Bindings.Sources {
			parts := strings.Split(s, "_")
			channels[c].Bindings.Sources[i] = parts[0]
		}
		for i, e := range channels[c].Bindings.Executors {
			parts := strings.Split(e, "_")
			channels[c].Bindings.Executors[i] = parts[0]
		}
	}
}
