//go:build integration

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"botkube.io/botube/test/botkubex"
	"botkube.io/botube/test/commplatform"
	"botkube.io/botube/test/diff"
	"botkube.io/botube/test/fake"
	"github.com/MakeNowJust/heredoc"
	"github.com/anthhub/forwarder"
	"github.com/hasura/go-graphql-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	netapiv1 "k8s.io/api/networking/v1"
	rbacapiv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	netv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/httpx"
	"github.com/kubeshop/botkube/pkg/ptr"
)

type ConfigProvider struct {
	Endpoint             string
	ApiKey               string
	SlackWorkspaceTeamID string
	ImageRepository      string `envconfig:"default=kubeshop/pr/botkube"`
	ImageRegistry        string `envconfig:"default=ghcr.io"`
	ImageTag             string
	HelmRepoDirectory    string
	BotkubeCliBinaryPath string

	Timeout time.Duration `envconfig:"default=15s"`
}

type Config struct {
	KubeconfigPath string `envconfig:"optional,KUBECONFIG"`
	Deployment     struct {
		Name          string        `envconfig:"default=botkube"`
		Namespace     string        `envconfig:"default=botkube"`
		ContainerName string        `envconfig:"default=botkube"`
		WaitTimeout   time.Duration `envconfig:"default=3m"`
		Envs          struct {
			SlackEnabledName              string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_ENABLED"`
			DefaultSlackChannelIDName     string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_CHANNELS_DEFAULT_NAME"`
			SecondarySlackChannelIDName   string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_CHANNELS_SECONDARY_NAME"`
			ThirdSlackChannelIDName       string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_CHANNELS_THIRD_NAME"`
			DiscordEnabledName            string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_ENABLED"`
			DefaultDiscordChannelIDName   string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_CHANNELS_DEFAULT_ID"`
			SecondaryDiscordChannelIDName string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_CHANNELS_SECONDARY_ID"`
			ThirdDiscordChannelIDName     string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_CHANNELS_THIRD_ID"`
			TeamsEnabledName              string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_TEAMS_ENABLED"`
			DefaultTeamsChannelIDName     string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_TEAMS_CHANNELS_DEFAULT_ID"`
			SecondaryTeamsChannelIDName   string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_TEAMS_CHANNELS_SECONDARY_ID"`
			ThirdTeamsChannelIDName       string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_TEAMS_CHANNELS_THIRD_ID"`
			BotkubePluginRepoURL          string `envconfig:"default=BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL"`
			LabelActionEnabledName        string `envconfig:"default=BOTKUBE_ACTIONS_LABEL-CREATED-SVC-RESOURCE_ENABLED"`
			StandaloneActionEnabledName   string `envconfig:"default=BOTKUBE_ACTIONS_GET-CREATED-RESOURCE_ENABLED"`
		}
	}
	IncomingWebhookService struct {
		Name      string `envconfig:"default=botkube"`
		Namespace string `envconfig:"default=botkube"`
		Port      int    `envconfig:"default=2115"`
		LocalPort int    `envconfig:"default=2115"`
	}
	Plugins   fake.PluginConfig
	ConfigMap struct {
		Namespace string `envconfig:"default=botkube"`
	}
	ClusterName      string `envconfig:"default=sample"`
	Slack            commplatform.SlackConfig
	Discord          commplatform.DiscordConfig
	Teams            commplatform.TeamsConfig
	ConfigProvider   ConfigProvider
	ShortWaitTimeout time.Duration `envconfig:"default=7s"`
}

const (
	testConfigMapName = "cm-watcher-trigger"
	// In cloud-based tests, after resource change in cloud, we can see extra messages as follows;
	// 1. Brace yourselves, incoming notifications from cluster '{name}'.
	// 2. Configuration reload requested for cluster '{name}'. Hold on a sec...
	// 3. My watch has ended for cluster '{name}'. See you soon!
	// 4. My watch begins for cluster '{name}'! :crossed_swords:
	// 5. Newer version (v1.7.0) of Botkube is available :tada:. Please upgrade Botkube backend.
	// Which means, we need to wait for 5 messages in total.
	limitLastMessageAfterCloudReload = 5
)

var (
	discordInvalidCmd = heredoc.Doc(`
				You must specify the type of resource to get. Use "kubectl api-resources" for a complete list of supported resources.

				error: Required resource not specified.
				Use "kubectl explain <resource>" for a detailed description of that resource (e.g. kubectl explain pods).
				See 'kubectl get -h' for help and examples

				exit status 1`)
	slackInvalidCmd = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(discordInvalidCmd)
	teamsInvalidCmd = discordInvalidCmd
	configMapLabels = map[string]string{
		"test.botkube.io": "true",
	}
	aliases = [][]string{
		{"kgp", "Get Pods", "kubectl get pods"},
		{"kgda", "Get Deployments", "kubectl get deployments -A"},
		{"e", "", "echo"},
		{"p", "", "ping"},
	}
)

func TestSlack(t *testing.T) {
	return
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	runBotTest(t,
		appCfg,
		commplatform.SlackBot,
		slackInvalidCmd,
		appCfg.Deployment.Envs.DefaultSlackChannelIDName,
		appCfg.Deployment.Envs.SecondarySlackChannelIDName,
		appCfg.Deployment.Envs.ThirdSlackChannelIDName,
	)
}

func TestDiscord(t *testing.T) {
	return
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	runBotTest(t,
		appCfg,
		commplatform.DiscordBot,
		discordInvalidCmd,
		appCfg.Deployment.Envs.DefaultDiscordChannelIDName,
		appCfg.Deployment.Envs.SecondaryDiscordChannelIDName,
		appCfg.Deployment.Envs.ThirdDiscordChannelIDName,
	)
}

func TestTeams(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	runBotTest(t,
		appCfg,
		commplatform.TeamsBot,
		teamsInvalidCmd,
		appCfg.Deployment.Envs.DefaultTeamsChannelIDName,
		appCfg.Deployment.Envs.SecondaryTeamsChannelIDName,
		appCfg.Deployment.Envs.ThirdTeamsChannelIDName,
	)
}

func newBotDriver(cfg Config, driverType commplatform.DriverType) (commplatform.BotDriver, error) {
	switch driverType {
	case commplatform.SlackBot:
		return commplatform.NewSlackTester(cfg.Slack, ptr.FromType(cfg.ConfigProvider.ApiKey))
	case commplatform.DiscordBot:
		return commplatform.NewDiscordTester(cfg.Discord)
	case commplatform.TeamsBot:
		return commplatform.NewTeamsTester(cfg.Teams, ptr.FromType(cfg.ConfigProvider.ApiKey))
	}
	return nil, nil
}

func runBotTest(t *testing.T,
	appCfg Config,
	driverType commplatform.DriverType,
	invalidCmdTemplate,
	deployEnvChannelIDName,
	deployEnvSecondaryChannelIDName,
	deployEnvRbacChannelIDName string,
) {
	botkubeDeploymentUninstalled := false
	t.Logf("Creating API client with provided token for %s...", driverType)
	botDriver, err := newBotDriver(appCfg, driverType)
	require.NoError(t, err)

	t.Log("Creating K8s client...")
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", appCfg.KubeconfigPath)
	require.NoError(t, err)
	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(t, err)

	var indexEndpoint string
	if botDriver.Type() == commplatform.DiscordBot {
		t.Log("Starting plugin server...")
		endpoint, startServerFn := fake.NewPluginServer(appCfg.Plugins)
		indexEndpoint = endpoint
		go func() {
			require.NoError(t, startServerFn())
		}()
	}

	t.Logf("Setting up test %s setup...", driverType)
	botDriver.InitUsers(t)

	botDriver.InitChannels(t)
	//cleanUpChannels := botDriver.InitChannels(t)
	//for _, fn := range cleanUpChannels {
	//	t.Cleanup(fn)
	//}

	channels := map[string]commplatform.Channel{
		deployEnvChannelIDName:          botDriver.Channel(),
		deployEnvSecondaryChannelIDName: botDriver.SecondChannel(),
		deployEnvRbacChannelIDName:      botDriver.ThirdChannel(),
	}

	for _, currentChannel := range channels {
		botDriver.PostInitialMessage(t, currentChannel.Identifier())
		botDriver.InviteBotToChannel(t, currentChannel.ID())
	}
	switch botDriver.Type() {
	case commplatform.DiscordBot:
		t.Log("Patching Deployment with test env variables...")
		deployNsCli := k8sCli.AppsV1().Deployments(appCfg.Deployment.Namespace)
		revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, botDriver.Type(), channels, indexEndpoint)
		t.Cleanup(func() { revertDeployFn(t) })

		t.Log("Waiting for Deployment")
		err = waitForDeploymentReady(deployNsCli, appCfg.Deployment.Name, appCfg.Deployment.WaitTimeout)
		require.NoError(t, err)
	case commplatform.SlackBot:
		t.Log("Creating Botkube Cloud instance...")
		gqlCli := NewClientForAPIKey(appCfg.ConfigProvider.Endpoint, appCfg.ConfigProvider.ApiKey)
		appCfg.ClusterName = botDriver.Channel().Name()
		deployment := gqlCli.MustCreateBasicDeploymentWithCloudSlack(t, appCfg.ClusterName, appCfg.ConfigProvider.SlackWorkspaceTeamID, botDriver.Channel().Name(), botDriver.SecondChannel().Name(), botDriver.ThirdChannel().Name())
		for _, alias := range aliases {
			gqlCli.MustCreateAlias(t, alias[0], alias[1], alias[2], deployment.ID)
		}
		t.Cleanup(func() {
			// We have a glitch on backend side and the logic below is a workaround for that.
			// Tl;dr uninstalling Helm chart reports "DISCONNECTED" status, and deployment deletion reports "DELETED" status.
			// If we do these two things too quickly, we'll run into resource version mismatch in repository logic.
			// Read more here: https://github.com/kubeshop/botkube-cloud/pull/486#issuecomment-1604333794
			for !botkubeDeploymentUninstalled {
				t.Log("Waiting for Helm chart uninstallation, in order to proceed with deleting Botkube Cloud instance...")
				time.Sleep(1 * time.Second)
			}

			t.Log("Helm chart uninstalled. Waiting a bit...")
			time.Sleep(3 * time.Second) // ugly, but at least we will be pretty sure we won't run into the resource version mismatch

			t.Log("Deleting Botkube Cloud instance...")
			gqlCli.MustDeleteDeployment(t, graphql.ID(deployment.ID))
		})

		err = botkubex.Install(t, botkubex.InstallParams{
			BinaryPath:                              appCfg.ConfigProvider.BotkubeCliBinaryPath,
			HelmRepoDirectory:                       appCfg.ConfigProvider.HelmRepoDirectory,
			ConfigProviderEndpoint:                  appCfg.ConfigProvider.Endpoint,
			ConfigProviderIdentifier:                deployment.ID,
			ConfigProviderAPIKey:                    deployment.APIKey.Value,
			ImageTag:                                appCfg.ConfigProvider.ImageTag,
			ImageRegistry:                           appCfg.ConfigProvider.ImageRegistry,
			ImageRepository:                         appCfg.ConfigProvider.ImageRepository,
			PluginRestartPolicyThreshold:            1,
			PluginRestartHealthCheckIntervalSeconds: 2,
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			t.Log("Uninstalling Helm chart...")
			botkubex.Uninstall(t, appCfg.ConfigProvider.BotkubeCliBinaryPath)
			botkubeDeploymentUninstalled = true
		})
	case commplatform.TeamsBot:
		t.Log("Creating Botkube Cloud instance...")
		gqlCli := NewClientForAPIKey(appCfg.ConfigProvider.Endpoint, appCfg.ConfigProvider.ApiKey)
		appCfg.ClusterName = botDriver.Channel().Name()
		deployment := gqlCli.MustCreateBasicDeploymentWithCloudTeams(t, appCfg.ClusterName, appCfg.Teams.OrganizationTeamID, botDriver.Channel().ID(), botDriver.SecondChannel().ID(), botDriver.ThirdChannel().ID())
		for _, alias := range aliases {
			gqlCli.MustCreateAlias(t, alias[0], alias[1], alias[2], deployment.ID)
		}
		t.Cleanup(func() {
			// We have a glitch on backend side and the logic below is a workaround for that.
			// Tl;dr uninstalling Helm chart reports "DISCONNECTED" status, and deployment deletion reports "DELETED" status.
			// If we do these two things too quickly, we'll run into resource version mismatch in repository logic.
			// Read more here: https://github.com/kubeshop/botkube-cloud/pull/486#issuecomment-1604333794
			for !botkubeDeploymentUninstalled {
				t.Log("Waiting for Helm chart uninstallation, in order to proceed with deleting Botkube Cloud instance...")
				time.Sleep(1 * time.Second)
			}

			t.Log("Helm chart uninstalled. Waiting a bit...")
			time.Sleep(3 * time.Second) // ugly, but at least we will be pretty sure we won't run into the resource version mismatch

			t.Log("Deleting Botkube Cloud instance...")
			gqlCli.MustDeleteDeployment(t, graphql.ID(deployment.ID))
		})

		err = botkubex.Install(t, botkubex.InstallParams{
			BinaryPath:                              appCfg.ConfigProvider.BotkubeCliBinaryPath,
			HelmRepoDirectory:                       appCfg.ConfigProvider.HelmRepoDirectory,
			ConfigProviderEndpoint:                  appCfg.ConfigProvider.Endpoint,
			ConfigProviderIdentifier:                deployment.ID,
			ConfigProviderAPIKey:                    deployment.APIKey.Value,
			ImageTag:                                appCfg.ConfigProvider.ImageTag,
			ImageRegistry:                           appCfg.ConfigProvider.ImageRegistry,
			ImageRepository:                         appCfg.ConfigProvider.ImageRepository,
			PluginRestartPolicyThreshold:            1,
			PluginRestartHealthCheckIntervalSeconds: 2,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			t.Log("Uninstalling Helm chart...")
			//botkubex.Uninstall(t, appCfg.ConfigProvider.BotkubeCliBinaryPath)
			botkubeDeploymentUninstalled = true
		})
	}

	cmdHeader := func(command string) string {
		return fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName)
	}

	// TODO: configure and use MessageWaitTimeout from an (app) Config as it targets both Slack and Discord.
	// Discord bot needs a bit more time to connect to Discord API.
	time.Sleep(appCfg.Discord.MessageWaitTimeout)
	t.Log("Waiting for interactive help")
	expMessage := interactive.NewHelpMessage(config.CommPlatformIntegration(botDriver.Type()), appCfg.ClusterName, []string{"botkube/helm", "botkube/kubectl"}).Build()
	botDriver.ReplaceBotNamePlaceholder(&expMessage, appCfg.ClusterName)
	err = botDriver.WaitForInteractiveMessagePostedRecentlyEqual(botDriver.BotUserID(),
		botDriver.Channel().ID(),
		expMessage,
	)
	require.NoError(t, err)

	t.Log("Waiting for Bot message in channel...")
	err = botDriver.WaitForMessagePostedRecentlyEqual(botDriver.BotUserID(), botDriver.Channel().ID(), fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName))
	require.NoError(t, err)

	t.Log("Running actual test cases")

	t.Run("Ping", func(t *testing.T) {
		aliasedCommand := "p"
		expandedCommand := "ping"
		expectedMessage := fmt.Sprintf("`%s` on `%s`\n```\npong", expandedCommand, appCfg.ClusterName)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCommand)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Help", func(t *testing.T) {
		command := "help"
		expectedMessage := interactive.NewHelpMessage(config.CommPlatformIntegration(botDriver.Type()), appCfg.ClusterName, []string{"botkube/helm", "botkube/kubectl"}).Build()
		botDriver.ReplaceBotNamePlaceholder(&expectedMessage, appCfg.ClusterName)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastInteractiveMessagePostedEqual(botDriver.BotUserID(),
			botDriver.Channel().ID(),
			expectedMessage,
		)

		assert.NoError(t, err)
	})

	t.Run("Botkube PluginManagement", func(t *testing.T) {
		t.Run("Echo Executor success", func(t *testing.T) {
			command := "echo test"
			expectedBody := codeBlock(strings.ToUpper(command))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})
		t.Run("Echo Executor success using alias", func(t *testing.T) {
			aliasedCommand, expandedCommand := cmdWithAliasPrefix(aliasedCmd{
				expandedPrefix: "echo", aliasedPrefix: "e", cmd: "alias",
			})
			expectedBody := codeBlock(strings.ToUpper("echo alias"))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(expandedCommand), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCommand)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})
		t.Run("Echo Executor failure", func(t *testing.T) {
			command := "echo @fail"
			expectedBody := codeBlock("The @fail label was specified. Failing execution.")
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Echo Executor help", func(t *testing.T) {
			command := "echo help"
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

			expectedBody := ".... empty response _*<cricket sounds>*_ :cricket: :cricket: :cricket:"
			if botDriver.Type() == commplatform.SlackBot {
				expectedBody = ".... empty response _*&lt;cricket sounds&gt;*_ :cricket: :cricket: :cricket:"
			}

			err = waitForLastPlaintextMessageWithHeaderEqual(appCfg, botDriver, command, expectedBody)
			assert.NoError(t, err)
		})

		t.Run("Helm Executor", func(t *testing.T) {
			command := "helm install --help"
			expectedBody := codeBlock(heredoc.Doc(`
				Installs a chart archive.

				There are two different ways you to install a Helm chart:
				1. By absolute URL: helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
				2. By chart reference and repo url: helm install --repo https://example.com/charts/ mynginx nginx

				Usage:
				    helm install [NAME] [CHART] [flags]

				Flags:
				    --create-namespace
				    --generate-name,-g
				    --dependency-update
				    --description
				    --devel
				    --disable-openapi-validation
				    --dry-run
				    --insecure-skip-tls-verify
				    --name-template
				    --no-hooks
				    --pass-credentials
				    --password
				    --post-renderer
				    --post-renderer-args
				    --render-subchart-notes
				    --replace
				    --repo
				    --set
				    --set-json
				    --set-string
				    --skip-crds
				    --timeout
				    --username
				    --verify
				    --version
				    -o,--output`))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
			if botDriver.Type() == commplatform.SlackBot {
				expectedMessage = fmt.Sprintf("%s %s", cmdHeader(command), expectedBody)
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Helm Executor help", func(t *testing.T) {
			command := "helm help"
			expectedBody := codeBlock(heredoc.Doc(`
				The official Botkube plugin for the Helm CLI.

				Usage:
				  helm [command]

				Available Commands:
				  install     # Installs a given chart to cluster where Botkube is installed.
				  list        # Lists all releases on cluster where Botkube is installed.
				  rollback    # Rolls back a given release to a previous revision.
				  status      # Displays the status of the named release.
				  test        # Runs tests for a given release.
				  uninstall   # Uninstalls a given release.
				  upgrade     # Upgrades a given release.
				  version     # Shows the version of the Helm CLI used by this Botkube plugin.
				  history     # Shows release history
				  get         # Shows extended information of a named release

				Flags:
				    --namespace,-n
				    --debug
				    --burst-limit

				Use "helm [command] --help" for more information about the command.`))

			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
			if botDriver.Type() == commplatform.SlackBot {
				expectedMessage = fmt.Sprintf("%s %s", cmdHeader(command), expectedBody)
			}
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("ConfigMap watcher source streaming", func(t *testing.T) {
			t.Log("Creating sample ConfigMap...")
			cfgMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-watcher-trigger",
					Namespace: appCfg.Deployment.Namespace,
					// for now, it allows us to disable the built-in kubernetes source and make sure that
					// only the plugged one will respond
					Annotations: map[string]string{
						"botkube.io/disable": "true",
					},
				},
			}

			cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
			cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, nil) })

			t.Log("Expecting bot message channel...")
			expectedMsg := fmt.Sprintf("Plugin cm-watcher detected `ADDED` event on `%s/%s`", cfgMap.Namespace, cfgMap.Name)

			err = waitForLastPlaintextMessageEqual(botDriver, botDriver.Channel().ID(), expectedMsg)
			assert.NoError(t, err)
		})

		t.Run("ConfigMap watcher source external requests", func(t *testing.T) {
			t.Logf("Setting up port forwarding for %s/%s service...", appCfg.IncomingWebhookService.Namespace, appCfg.IncomingWebhookService.Name)
			t.Logf("Using local port %d and remote port %d...", appCfg.IncomingWebhookService.LocalPort, appCfg.IncomingWebhookService.Port)
			options := []*forwarder.Option{
				{
					LocalPort:   appCfg.IncomingWebhookService.LocalPort,
					RemotePort:  appCfg.IncomingWebhookService.Port,
					Source:      fmt.Sprintf("svc/%s", appCfg.IncomingWebhookService.Name),
					ServiceName: appCfg.IncomingWebhookService.Name,
					Namespace:   appCfg.IncomingWebhookService.Namespace,
				},
			}
			ret, err := forwarder.WithForwarders(context.Background(), options, appCfg.KubeconfigPath)
			require.NoError(t, err)
			defer ret.Close()

			t.Log("Waiting for port forwarding to be ready...")
			_, err = ret.Ready()
			require.NoError(t, err)

			sourceName := "other-plugins"
			t.Logf("Sending a request to the incoming webhook to trigger the %s source plugin...", sourceName)
			message := "Hello there!"
			sendIncomingWebhookRequest(t, appCfg.IncomingWebhookService.LocalPort, sourceName, message)

			t.Log("Expecting bot message channel...")
			expectedMsg := fmt.Sprintf("*Incoming webhook event:* %s", message)

			err = waitForLastPlaintextMessageEqual(botDriver, botDriver.Channel().ID(), expectedMsg)
			assert.NoError(t, err)
		})
	})

	t.Run("Show config", func(t *testing.T) {
		t.Run("With custom cluster name and filter", func(t *testing.T) {
			command := fmt.Sprintf("show config --filter=cacheDir --cluster-name %s", appCfg.ClusterName)
			expectedFilteredBody := codeBlock(heredoc.Doc(`cacheDir: /tmp`))
			expectedMessage := fmt.Sprintf("`show config --filter=cacheDir --cluster-name %s` on `%s`\n%s", appCfg.ClusterName, appCfg.ClusterName, expectedFilteredBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With filter", func(t *testing.T) {
			command := "show config --filter=cacheDir"
			expectedFilteredBody := codeBlock(heredoc.Doc(`cacheDir: /tmp`))
			expectedMessage := fmt.Sprintf("`show config --filter=cacheDir` on `%s`\n%s", appCfg.ClusterName, expectedFilteredBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With unknown cluster name", func(t *testing.T) {
			command := "show config --cluster-name non-existing"

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			expMessage := "Instance not found"
			userId := botDriver.BotUserID()

			if botDriver.Type() == commplatform.DiscordBot {
				t.Log("Ensuring bot didn't post anything new...")
				time.Sleep(appCfg.Slack.MessageWaitTimeout)
				expMessage = command
				userId = botDriver.TesterUserID()
			}

			// Same expected message as before
			err = botDriver.WaitForLastMessageContains(userId, botDriver.Channel().ID(), expMessage)
			assert.NoError(t, err)
		})
	})

	t.Run("Executor", func(t *testing.T) {
		hasValidHeader := func(cmd, msg string) bool {
			if botDriver.Type() == commplatform.TeamsBot {
				// Teams uses AdaptiveCard and the built-in table format, that's the reason why we can't
				// compare it with the plain text message. On the other hand, comparing JSON format would require us
				// to normalize the table cells (e.g. time)

				if strings.HasPrefix(msg, "{") {
					cmd = strconv.Quote(cmd) // it is a JSON so it will be escaped
				}
				// message is in JSON
				return strings.Contains(msg, cmd) &&
					strings.Contains(msg, " on ") &&
					strings.Contains(msg, appCfg.ClusterName)
			}
			return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", cmd, appCfg.ClusterName)))
		}

		t.Run("Get Deployment", func(t *testing.T) {
			command := fmt.Sprintf("kubectl get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return hasValidHeader(command, msg) && strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Deployment with matching filter", func(t *testing.T) {
			command := fmt.Sprintf(`kubectl get deploy -n %s %s --filter='botkube'`, appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return hasValidHeader(command, msg) && strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("kubectl get configmap -n %s", appCfg.Deployment.Namespace)
			assertConfigMaps := func(msg string) bool {
				return strings.Contains(msg, "kube-root-ca.crt") && strings.Contains(msg, "botkube-global-config")
			}

			if botDriver.Type().IsCloud() {
				assertConfigMaps = func(msg string) bool {
					return strings.Contains(msg, "kube-root-ca.crt")
				}
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, func(msg string) (bool, int, string) {
				return hasValidHeader(command, msg) && assertConfigMaps(msg), 0, ""
			})
			assert.NoError(t, err)
		})

		t.Run("Get Configmap with mismatching filter", func(t *testing.T) {
			command := fmt.Sprintf(`kubectl get configmap -n %s --filter='unknown-thing'`, appCfg.Deployment.Namespace)
			assertionFn := func(msg string) (bool, int, string) {
				return hasValidHeader(command, msg) &&
					!strings.Contains(msg, "kube-root-ca.crt") &&
					!strings.Contains(msg, "botkube-global-config"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Receive large output as plaintext file with executor command as message", func(t *testing.T) {
			if botDriver.Type() == commplatform.TeamsBot {
				t.Skip() // FIXME: https://github.com/kubeshop/botkube-cloud/issues/728
			}
			command := fmt.Sprintf("kubectl get pod -o yaml -n %s", appCfg.Deployment.Namespace)
			fileUploadAssertionFn := func(title, mimetype string) bool {
				return title == "Response.txt" && strings.Contains(mimetype, "text/plain")
			}
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePostedWithFileUpload(botDriver.BotUserID(), botDriver.Channel().ID(), fileUploadAssertionFn)
			assert.NoError(t, err)

			assertionFn := func(msg string) (bool, int, string) {
				return hasValidHeader(command, msg), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
		})

		t.Run("Get forbidden resource", func(t *testing.T) {
			command := "kubectl get role"
			expectedBody := heredoc.Docf(`
				Error from server (Forbidden): roles.rbac.authorization.k8s.io is forbidden: User "kubectl-first-channel" cannot list resource "roles" in API group "rbac.authorization.k8s.io" in the namespace "default"

				exit status 1`)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

			err := waitForLastCodeBlockMessageWithHeaderEqual(appCfg, botDriver, command, expectedBody)
			assert.NoError(t, err)
		})
		t.Run("Specify unknown command", func(t *testing.T) {
			command := "unknown"
			expectedBody := codeBlock("Command not supported. Please use 'help' to see supported commands.")
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify invalid command", func(t *testing.T) {
			command := "kubectl get"

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

			err := waitForLastCodeBlockMessageWithHeaderEqual(appCfg, botDriver, command, invalidCmdTemplate)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "kubectl get po --namespace team-b"
			expectedBody := heredoc.Docf(`
				Error from server (Forbidden): pods is forbidden: User "kubectl-first-channel" cannot list resource "pods" in API group "" in the namespace "team-b"

				exit status 1`)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

			err := waitForLastCodeBlockMessageWithHeaderEqual(appCfg, botDriver, command, expectedBody)
			assert.NoError(t, err)
		})

		t.Run("Based on other bindings", func(t *testing.T) {
			t.Run("Wait for Deployment", func(t *testing.T) {
				command := fmt.Sprintf("kubectl wait deployment -n %s %s --for condition=Available=True", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
				expectedBody := codeBlock(`The "wait" command is not supported by the Botkube kubectl plugin.`)
				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Exec (the kubectl which is disabled)", func(t *testing.T) {
				command := fmt.Sprintf("kubectl exec deploy/%s -n %s -- date", appCfg.Deployment.Name, appCfg.Deployment.Namespace)
				expectedBody := codeBlock(heredoc.Docf(`
				Error from server (Forbidden): pods "botkube-pod" is forbidden: User "kubectl-first-channel" cannot create resource "pods/exec" in API group "" in the namespace "botkube"

				exit status 1`))
				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

				podName, err := regexp.Compile(`"botkube-.*-.*" is`)
				assert.NoError(t, err)

				assertionFn := func(msg string) (bool, int, string) {
					msg = podName.ReplaceAllString(msg, `"botkube-pod" is`)

					switch botDriver.Type() {
					case commplatform.TeamsBot:
						msg, expectedMessage = commplatform.NormalizeTeamsWhitespacesInMessages(msg, expectedMessage)
					default:
						msg = commplatform.TrimSlackMsgTrailingLine(msg)
					}

					if !strings.EqualFold(expectedMessage, msg) {
						count := diff.CountMatchBlock(expectedMessage, msg)
						msgDiff := diff.Diff(expectedMessage, msg)
						return false, count, msgDiff
					}
					return true, 0, ""
				}
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})

			t.Run("Get all Pods with alias", func(t *testing.T) {
				aliasedCommand := "kgp -A"
				expandedCommand := "kubectl get pods -A"

				expectedBody := heredoc.Docf(`
				Error from server (Forbidden): pods is forbidden: User "kubectl-first-channel" cannot list resource "pods" in API group "" at the cluster scope

				exit status 1`)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCommand)
				err := waitForLastCodeBlockMessageWithHeaderEqual(appCfg, botDriver, expandedCommand, expectedBody)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments with alias", func(t *testing.T) {
				aliasedCommand := "kgda"
				expandedCommand := "kubectl get deployments -A"
				assertionFn := func(msg string) (bool, int, string) {
					return hasValidHeader(expandedCommand, msg) &&
						strings.Contains(msg, "local-path-provisioner") &&
						strings.Contains(msg, "coredns") &&
						strings.Contains(msg, "botkube"), 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCommand)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})
		})

		k8sPrefixTests := []string{"kubectl", "kc", "k"}
		for _, prefix := range k8sPrefixTests {
			t.Run(fmt.Sprintf("Get Pods with k8s prefix %s", prefix), func(t *testing.T) {
				aliasedCmd, expandedCmd := kubectlAliasedCommand(prefix, fmt.Sprintf("get pods --namespace %s", appCfg.Deployment.Namespace))
				assertionFn := func(msg string) (bool, int, string) {
					return hasValidHeader(expandedCmd, msg) &&
						hasAllColumns(msg, "NAME", "READY", "STATUS", "RESTART", "AGE"), 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCmd)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})
		}
	})

	var firstCMUpdate commplatform.ExpAttachmentInput
	t.Run("Multi-channel notifications", func(t *testing.T) {
		t.Log("Getting notifier status from second channel...")
		command := "status notifications"
		expectedBody := codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))

		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)

		if botDriver.Type() == commplatform.TeamsBot {
			// TODO(add option to configure notifications): https://github.com/kubeshop/botkube-cloud/issues/841
			expectedBody = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are enabled here.", appCfg.ClusterName))
		}

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Starting notifier in second channel...")
		command = "enable notifications"
		expectedBody = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)

		limitMessages := 1
		if botDriver.Type().IsCloud() {
			// Which means, we need to wait for 5 messages in total.
			// 1. Brace yourselves, incoming notifications from cluster '{name}'.
			// 2. Configuration reload requested for cluster '{name}'. Hold on a sec...
			// 3. My watch has ended for cluster '{name}'. See you soon!
			// 4. My watch begins for cluster '{name}'! :crossed_swords:
			// 5. Newer version (v1.7.0) of Botkube is available :tada:. Please upgrade Botkube backend.
			// Which means, we need to wait for 5 messages in total.
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.SecondChannel().ID(), limitMessages, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)

		t.Log("Creating ConfigMap...")
		var cfgMapAlreadyDeleted bool
		cfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      botDriver.Channel().Name(),
				Namespace: appCfg.Deployment.Namespace,
				Labels:    configMapLabels,
			},
		}

		createCMEventTime := time.Now()
		cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })

		t.Log("Expecting bot message in first channel...")
		expAttachmentIn := commplatform.ExpAttachmentInput{
			AllowedTimestampDelta: time.Minute,
			Message: api.Message{
				Type:      api.NonInteractiveSingleSection,
				Timestamp: createCMEventTime,
				Sections: []api.Section{
					{
						Base: api.Base{
							Header: "🟢 v1/configmaps created",
						},
						TextFields: api.TextFields{
							{Key: "Kind", Value: "ConfigMap"},
							{Key: "Name", Value: cfgMap.Name},
							{Key: "Namespace", Value: cfgMap.Namespace},
							{Key: "Cluster", Value: appCfg.ClusterName},
						},
					},
				},
			},
		}
		limitMessages = 2
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, expAttachmentIn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new in second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		limitMessages = 2
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.SecondChannel().ID(), 5, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		t.Log("Updating ConfigMap for not watched field...")
		cfgMap.Annotations = map[string]string{
			"my": "annotation",
		}
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		limitMessages = 2
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, expAttachmentIn)
		require.NoError(t, err)
		limitMessages = 2
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.SecondChannel().ID(), 5, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		t.Log("Updating ConfigMap for observed field...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		updateCMEventTime := time.Now()
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message in all channels...")
		// Third (RBAC) channel is isolated from this
		channelIDs := []string{channels[deployEnvChannelIDName].ID(), channels[deployEnvSecondaryChannelIDName].ID()}
		for _, channelID := range channelIDs {
			err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), channelID, 2, commplatform.ExpAttachmentInput{
				AllowedTimestampDelta: time.Minute,
				Message: api.Message{
					Type:      api.NonInteractiveSingleSection,
					Timestamp: updateCMEventTime,
					Sections: []api.Section{
						{
							Base: api.Base{
								Header: "💡 v1/configmaps updated",
							},
							TextFields: api.TextFields{
								{Key: "Kind", Value: "ConfigMap"},
								{Key: "Name", Value: cfgMap.Name},
								{Key: "Namespace", Value: cfgMap.Namespace},
								{Key: "Cluster", Value: appCfg.ClusterName},
							},
						},
					},
				},
			})
			require.NoError(t, err)
		}

		t.Log("Stopping notifier in first channel...")
		command = "disable notifications"
		expectedBody = codeBlock(fmt.Sprintf("Sure! I won't send you notifications from cluster '%s' here.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		limitMessages = 1
		if botDriver.Type().IsCloud() {
			waitForRestart(t, botDriver, botDriver.BotUserID(), botDriver.Channel().ID(), appCfg.ClusterName)
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 5, botDriver.AssertEquals(expectedMessage))
		assert.NoError(t, err)

		t.Log("Getting notifier status from second channel...")
		command = "status notifications"
		expectedBody = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are enabled here.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from first channel...")
		command = "status notifications"
		expectedBody = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Updating ConfigMap once again...")
		updateCMEventTime = time.Now()
		cfgMap.Data = map[string]string{
			"operation": "update-second",
		}
		_, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new on first channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		// Same expected message as before
		limitMessages = 1
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		secondCMUpdate := commplatform.ExpAttachmentInput{
			AllowedTimestampDelta: time.Minute,
			Message: api.Message{
				Type:      api.NonInteractiveSingleSection,
				Timestamp: updateCMEventTime,
				Sections: []api.Section{
					{
						Base: api.Base{
							Header: "💡 v1/configmaps updated",
						},
						TextFields: api.TextFields{
							{Key: "Kind", Value: "ConfigMap"},
							{Key: "Name", Value: cfgMap.Name},
							{Key: "Namespace", Value: cfgMap.Namespace},
							{Key: "Cluster", Value: appCfg.ClusterName},
						},
					},
				},
			},
		}
		t.Log("Expecting bot message in second channel...")
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), 2, secondCMUpdate)

		t.Log("Starting notifier in first channel")
		command = "enable notifications"
		expectedBody = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		limitMessages = 1
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		if botDriver.Type().IsCloud() {
			waitForRestart(t, botDriver, botDriver.BotUserID(), botDriver.Channel().ID(), appCfg.ClusterName)
		}

		t.Log("Creating and deleting ignored ConfigMap")
		ignoredCfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-ignored", botDriver.Channel().Name()),
				Namespace: appCfg.Deployment.Namespace,
				Annotations: map[string]string{
					"botkube.io/disable": "true",
				},
			},
		}
		_, err = cfgMapCli.Create(context.Background(), ignoredCfgMap, metav1.CreateOptions{})
		require.NoError(t, err)
		err = cfgMapCli.Delete(context.Background(), ignoredCfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		limitMessages = 1
		if botDriver.Type().IsCloud() {
			limitMessages = limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, botDriver.AssertEquals(expectedMessage))
		require.NoError(t, err)

		t.Log("Deleting ConfigMap")
		deleteCMEventTime := time.Now()
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		firstCMUpdate = commplatform.ExpAttachmentInput{
			AllowedTimestampDelta: time.Minute,
			Message: api.Message{
				Type:      api.NonInteractiveSingleSection,
				Timestamp: deleteCMEventTime,
				Sections: []api.Section{
					{
						Base: api.Base{
							Header: "❗ v1/configmaps deleted",
						},
						TextFields: api.TextFields{
							{Key: "Kind", Value: "ConfigMap"},
							{Key: "Name", Value: cfgMap.Name},
							{Key: "Namespace", Value: cfgMap.Namespace},
							{Key: "Cluster", Value: appCfg.ClusterName},
						},
					},
				},
			},
		}
		t.Log("Expecting bot message on first channel...")
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, firstCMUpdate)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new in second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		limitMessages = 2
		if botDriver.Type().IsCloud() {
			// There are 2 config reload requested after second cm update
			limitMessages = 2* limitLastMessageAfterCloudReload
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), limitMessages, secondCMUpdate)
		require.NoError(t, err)
	})

	t.Run("Recommendations and actions", func(t *testing.T) {
		podCli := k8sCli.CoreV1().Pods(appCfg.Deployment.Namespace)
		podDefaultNSCli := k8sCli.CoreV1().Pods("default")
		svcCli := k8sCli.CoreV1().Services(appCfg.Deployment.Namespace)

		t.Log("Creating Pod in namespace 'default'. This pod should not be included in recommendations...")
		podIgnored := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      botDriver.Channel().Name(),
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{Name: "nginx", Image: "nginx:latest"},
				},
			},
		}
		podIgnored, err = podDefaultNSCli.Create(context.Background(), podIgnored, metav1.CreateOptions{})
		require.NoError(t, err)
		t.Cleanup(func() { cleanupCreatedPod(t, podDefaultNSCli, podIgnored.Name) })

		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		limitMessages := 1
		if botDriver.Type().IsCloud() {
			limitMessages = 5
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), limitMessages, firstCMUpdate)
		require.NoError(t, err)

		t.Log("Creating Pod...")
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      botDriver.Channel().Name(),
				Namespace: appCfg.Deployment.Namespace,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{Name: "nginx", Image: "nginx:latest"},
				},
			},
		}
		require.Len(t, pod.Spec.Containers, 1)
		createPodEventTime := time.Now()
		pod, err = podCli.Create(context.Background(), pod, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedPod(t, podCli, pod.Name) })

		t.Log("Expecting bot event message...")
		// we check last 3 messages as we can get:
		// - message with recommendations from 'k8s-events'
		// - massage with pod create event from 'k8s-pod-create-events'
		// - message with kc execution via 'get-created-resource' automation
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 3, commplatform.ExpAttachmentInput{
			AllowedTimestampDelta: time.Minute,
			Message: api.Message{
				Type:      api.NonInteractiveSingleSection,
				Timestamp: createPodEventTime,
				Sections: []api.Section{
					{
						Base: api.Base{
							Header: "🟢 v1/pods created",
						},
						TextFields: api.TextFields{
							{Key: "Kind", Value: "Pod"},
							{Key: "Name", Value: pod.Name},
							{Key: "Namespace", Value: pod.Namespace},
							{Key: "Cluster", Value: appCfg.ClusterName},
						},
						BulletLists: []api.BulletList{
							{
								Title: "Recommendations",
								Items: []string{
									fmt.Sprintf("Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name),
									fmt.Sprintf("The 'latest' tag used in 'nginx:latest' image of Pod '%s/%s' container 'nginx' should be avoided.", pod.Namespace, pod.Name),
								},
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		t.Log("Expecting bot automation message...")
		hasValidHeaderWithAuthor := func(msg, command, author string) bool {
			if botDriver.Type() == commplatform.TeamsBot {
				// Teams uses AdaptiveCard and the built-in table format, that's the reason why we can't
				// compare it with the plain text message. On the other hand, comparing JSON format would require us
				// to normalize the table cells (e.g. time)
				// message is in JSON
				return strings.Contains(msg, command) &&
					strings.Contains(msg, " on ") &&
					strings.Contains(msg, appCfg.ClusterName) &&
					strings.Contains(msg, strconv.Quote(author))
			}

			return strings.Contains(msg, fmt.Sprintf("`%s` on `%s`%s", command, appCfg.ClusterName, author))
		}
		command := fmt.Sprintf(`kubectl get pod -n %s %s`, pod.Namespace, pod.Name)
		automationAssertionFn := func(msg string) (bool, int, string) {
			podNameCount := 2 // command + 1 occurrence in the command output
			if botDriver.Type().IsCloud() {
				podNameCount = 3 // command + on cluster name section + 1 occurrence in the command output
			}

			return hasValidHeaderWithAuthor(msg, command, " by Automation \"Get created resource\"") &&
					hasAllColumns(msg, "NAME", "READY", "STATUS") &&
					strings.Count(msg, pod.Name) == podNameCount,
				0, ""
		}
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 2, automationAssertionFn)
		require.NoError(t, err)

		t.Log("Creating Service...")
		svc := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      botDriver.Channel().Name(),
				Namespace: appCfg.Deployment.Namespace,
				Labels: map[string]string{
					"app": "e2e-test",
				},
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"app": "e2e-test",
				},
				Ports: []v1.ServicePort{
					{Port: 8080},
				},
			},
		}

		svc, err = svcCli.Create(context.Background(), svc, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedSvc(t, svcCli, svc.Name) })

		t.Log("Ensuring bot didn't post anything new on first channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		// same expected message as before
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 2, automationAssertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot automation was executed and label created Service...")
		err = wait.PollUntilContextTimeout(context.Background(), pollInterval, appCfg.Slack.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
			svc, err := svcCli.Get(context.Background(), svc.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			_, found := svc.GetLabels()["botkube-action"]
			return found, nil
		})
		assert.NoError(t, err, "while waiting for Service to be labeled by not bind automation")
	})

	t.Run("List actions", func(t *testing.T) {
		command := "list actions"
		expectedBody := codeBlock(heredoc.Doc(`
			ACTION                     ENABLED  DISPLAY NAME
			describe-created-resource  false    Describe created resource
			get-created-resource       true     Get created resource
			label-created-svc-resource true     Label created Service
			show-logs-on-error         false    Show logs on error`))

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("List executors", func(t *testing.T) {
		command := "list executors"
		expectedBody := codeBlock(heredoc.Doc(`
			EXECUTOR                   ENABLED ALIASES RESTARTS STATUS  LAST_RESTART
			botkube/echo@v0.0.0-latest true    e       0/1      Running 
			botkube/helm               true            0/1      Running 
			botkube/kubectl            true    k, kc   0/1      Running`))

		if botDriver.Type() == commplatform.TeamsBot {
			expectedBody = trimRightWhitespace(expectedBody)
		}
		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("List aliases", func(t *testing.T) {
		command := "list aliases"
		expectedBody := codeBlock(heredoc.Doc(`
			ALIAS COMMAND                    DISPLAY NAME
			e     echo                       
			k     kubectl                    Kubectl alias
			kc    kubectl                    Kubectl alias
			kgda  kubectl get deployments -A Get Deployments
			kgp   kubectl get pods           Get Pods
			p     ping`))
		contextMsg := "Only showing aliases for executors enabled for this channel."
		expectedMessage := fmt.Sprintf("%s\n\n%s\n%s", cmdHeader(command), expectedBody, contextMsg)
		if botDriver.Type() == commplatform.SlackBot {
			expectedMessage = fmt.Sprintf("%s %s %s", cmdHeader(command), expectedBody, contextMsg)
		}

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

		switch botDriver.Type() {
		case commplatform.SlackBot, commplatform.DiscordBot:
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)

		case commplatform.TeamsBot:
			// in this case of a plain text message, Teams renderer uses Adaptive Cards format
			// TODO(fix formatting for aliases table): https://github.com/kubeshop/botkube-cloud/issues/752#issuecomment-1908669638
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, func(msg string) (bool, int, string) {
				return hasAllColumns(msg, "ALIAS", "COMMAND", "DISPLAY NAME"), 0, ""
			})
			require.NoError(t, err)

		}
	})

	t.Run("List sources", func(t *testing.T) {
		command := "list sources"
		expectedBody := codeBlock(heredoc.Doc(`
			SOURCE             ENABLED RESTARTS STATUS  LAST_RESTART
			botkube/cm-watcher true    0/1      Running 
			botkube/kubernetes true    0/1      Running`))
		if botDriver.Type() == commplatform.TeamsBot {
			expectedBody = trimRightWhitespace(expectedBody)
		}

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("RBAC", func(t *testing.T) {
		t.Run("No configuration", func(t *testing.T) {
			echoParam := "john doe"
			command := fmt.Sprintf("echo %s", echoParam)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "JOHN DOE"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.ThirdChannel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.ThirdChannel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Default configuration", func(t *testing.T) {
			t.Log("Creating RBAC ConfigMap...")
			cfgMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-rbac",
					Namespace: appCfg.Deployment.Namespace,
					Labels: map[string]string{
						"rbac.botkube.io": "true",
					},
				},
			}

			cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
			cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
			require.NoError(t, err)

			var cfgMapAlreadyDeleted bool
			err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
			require.NoError(t, err)
			cfgMapAlreadyDeleted = true

			t.Log("Expecting bot message in third channel...")
			expectedMsg := fmt.Sprintf("Plugin cm-watcher detected `DELETED` event on `%s/%s`", cfgMap.Namespace, cfgMap.Name)

			err = waitForLastPlaintextMessageEqual(botDriver, botDriver.ThirdChannel().ID(), expectedMsg)
			require.NoError(t, err)

			t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })
		})

		t.Run("Static mapping", func(t *testing.T) {
			t.Log("Creating RBAC ConfigMap with Static mapping...")
			cfgMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-rbac-static",
					Namespace: appCfg.Deployment.Namespace,
					Annotations: map[string]string{
						"rbac.botkube.io": "true",
					},
				},
			}

			createCMEventTime := time.Now()
			cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
			cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Log("Expecting bot event message...")
			err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, commplatform.ExpAttachmentInput{
				AllowedTimestampDelta: time.Minute,
				Message: api.Message{
					Type:      api.NonInteractiveSingleSection,
					Timestamp: createCMEventTime,
					Sections: []api.Section{
						{
							Base: api.Base{
								Header: "🟢 v1/configmaps created",
							},
							TextFields: api.TextFields{
								{Key: "Kind", Value: "ConfigMap"},
								{Key: "Name", Value: cfgMap.Name},
								{Key: "Namespace", Value: cfgMap.Namespace},
								{Key: "Cluster", Value: appCfg.ClusterName},
							},
						},
					},
				},
			})
			require.NoError(t, err)

			cfgMapAlreadyDeleted := false

			t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })
		})

		t.Run("ChannelName mapping", func(t *testing.T) {
			clusterRole := &rbacapiv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: botDriver.ThirdChannel().Identifier(),
				},
				Rules: []rbacapiv1.PolicyRule{
					{
						APIGroups: []string{"networking.k8s.io"},
						Resources: []string{"ingresses"},
						Verbs:     []string{"get"},
					},
				},
			}

			t.Log("Creating RBAC ClusterRole for ChannelName mapping...")
			clusterRoleCli := k8sCli.RbacV1().ClusterRoles()
			cr, err := clusterRoleCli.Create(context.Background(), clusterRole, metav1.CreateOptions{})
			require.NoError(t, err)

			clusterRoleBinding := &rbacapiv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: botDriver.ThirdChannel().Identifier(),
				},
				RoleRef: rbacapiv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     botDriver.ThirdChannel().Identifier(),
				},
				Subjects: []rbacapiv1.Subject{
					{
						Kind:     "Group",
						Name:     botDriver.ThirdChannel().Name(),
						APIGroup: "rbac.authorization.k8s.io",
					},
				},
			}

			t.Log("Creating RBAC ClusterRoleBinding for ChannelName mapping...")
			clusterRoleBindingCli := k8sCli.RbacV1().ClusterRoleBindings()
			crb, err := clusterRoleBindingCli.Create(context.Background(), clusterRoleBinding, metav1.CreateOptions{})
			require.NoError(t, err)

			ing := &netapiv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ing-rbac-channel",
				},
				Spec: netapiv1.IngressSpec{
					DefaultBackend: &netapiv1.IngressBackend{
						Service: &netapiv1.IngressServiceBackend{
							Name: "test",
							Port: netapiv1.ServiceBackendPort{
								Number: int32(8080),
							},
						},
					},
				},
			}

			t.Log("Creating Ingress...")
			ingressCli := k8sCli.NetworkingV1().Ingresses(appCfg.Deployment.Namespace)
			ingress, err := ingressCli.Create(context.Background(), ing, metav1.CreateOptions{})
			require.NoError(t, err)

			command := fmt.Sprintf("kubectl get ing %s -n %s -o yaml", ingress.Name, ingress.Namespace)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "creationTimestamp:"), 0, ""
			}
			botDriver.PostMessageToBot(t, botDriver.ThirdChannel().Identifier(), command)

			t.Log("Expecting bot event message...")
			limitMessages := 1
			if botDriver.Type() == commplatform.TeamsBot {
				limitMessages = 2 // we sent in Teams the filter input as the separate message, but the main body will be in the N-1
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.ThirdChannel().ID(), limitMessages, assertionFn)
			assert.NoError(t, err)
			t.Cleanup(func() { cleanupCreatedIng(t, ingressCli, ingress.Name) })
			t.Cleanup(func() { cleanupCreatedClusterRole(t, clusterRoleCli, cr.Name) })
			t.Cleanup(func() { cleanupCreatedClusterRoleBinding(t, clusterRoleBindingCli, crb.Name) })
		})
	})

	t.Run("Plugin crash & recovery", func(t *testing.T) {
		t.Run("Crash config map source", func(t *testing.T) {
			cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
			crashConfigMapSourcePlugin(t, cfgMapCli)

			t.Log("Waiting for cm-watcher plugin to recover from panic...")
			time.Sleep(appCfg.ShortWaitTimeout)

			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: testConfigMapName,
				},
			}
			_, err := cfgMapCli.Create(context.Background(), cm, metav1.CreateOptions{})
			require.NoError(t, err)

			expectedMessage := fmt.Sprintf("Plugin cm-watcher detected `ADDED` event on `%s/%s`", appCfg.Deployment.Namespace, testConfigMapName)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, expectedMessage), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 3, assertionFn)
			require.NoError(t, err)

			err = cfgMapCli.Delete(context.Background(), testConfigMapName, metav1.DeleteOptions{})
			require.NoError(t, err)
		})

		t.Run("Crash echo executor", func(t *testing.T) {
			command := "echo @panic"
			expectedMessage := "error reading from server"

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, expectedMessage), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)

			t.Log("Waiting for echo plugin to recover from panic...")
			time.Sleep(appCfg.ShortWaitTimeout)

			command = "echo hello"
			expectedBody := codeBlock(strings.ToUpper(command))
			expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)

			command = "echo @panic"
			expectedMessage = "error reading from server"
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			assertionFn = func(msg string) (bool, int, string) {
				return strings.Contains(msg, expectedMessage), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)

			t.Log("Waiting for plugin manager to deactivate echo plugin...")
			time.Sleep(appCfg.ShortWaitTimeout)
			command = "list executors"
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

			assertionFn = func(msg string) (bool, int, string) {
				return strings.Contains(msg, "Deactivated"), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)

			command = "echo foo"
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			t.Log("Ensuring bot didn't post anything new...")

			assertionFn = func(msg string) (bool, int, string) {
				return strings.Contains(msg, command), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.TesterUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})
	})
}

type aliasedCmd struct {
	aliasedPrefix  string
	expandedPrefix string
	cmd            string
}

func cmdWithAliasPrefix(in aliasedCmd) (string, string) {
	return fmt.Sprintf("%s %s", in.aliasedPrefix, in.cmd), fmt.Sprintf("%s %s", in.expandedPrefix, in.cmd)
}

func kubectlAliasedCommand(prefix, cmd string) (string, string) {
	return cmdWithAliasPrefix(aliasedCmd{
		aliasedPrefix:  prefix,
		expandedPrefix: "kubectl",
		cmd:            cmd,
	})
}

func codeBlock(in string) string {
	return fmt.Sprintf("```\n%s\n```", in)
}

func cleanupCreatedCfgMapIfShould(t *testing.T, cfgMapCli corev1.ConfigMapInterface, name string, cfgMapAlreadyDeleted *bool) {
	if cfgMapAlreadyDeleted != nil && *cfgMapAlreadyDeleted {
		return
	}

	t.Log("Cleaning up created ConfigMap...")
	err := cfgMapCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedPod(t *testing.T, podCli corev1.PodInterface, name string) {
	t.Log("Cleaning up created Pod...")
	err := podCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedSvc(t *testing.T, podCli corev1.ServiceInterface, name string) {
	t.Log("Cleaning up created Service...")
	err := podCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}
func cleanupCreatedIng(t *testing.T, ingressCli netv1.IngressInterface, name string) {
	t.Log("Cleaning up created Ingress...")
	err := ingressCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedClusterRole(t *testing.T, clusterRoleCli rbacv1.ClusterRoleInterface, name string) {
	t.Log("Cleaning up created ClusterRole...")
	err := clusterRoleCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedClusterRoleBinding(t *testing.T, clusterRoleBindingCli rbacv1.ClusterRoleBindingInterface, name string) {
	t.Log("Cleaning up created ClusterRoleBinding...")
	err := clusterRoleBindingCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func sendIncomingWebhookRequest(t *testing.T, localPort int, sourceName, message string) {
	t.Helper()

	jsonBody := []byte(fmt.Sprintf(`{"message": "%s"}`, message))
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://localhost:%d/sources/v1/%s", localPort, sourceName),
		bytes.NewReader(jsonBody),
	)
	require.NoError(t, err)

	client := httpx.NewHTTPClient()
	res, err := client.Do(req)
	require.NoError(t, err)

	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func crashConfigMapSourcePlugin(t *testing.T, cfgMapCli corev1.ConfigMapInterface) {
	t.Helper()
	t.Log("Crashing ConfigMap source plugin...")
	_ = cfgMapCli.Delete(context.Background(), testConfigMapName, metav1.DeleteOptions{})

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: testConfigMapName,
			Annotations: map[string]string{
				"die": "true",
			},
		},
	}
	_, err := cfgMapCli.Create(context.Background(), cm, metav1.CreateOptions{})
	require.NoError(t, err)

	err = cfgMapCli.Delete(context.Background(), testConfigMapName, metav1.DeleteOptions{})
	require.NoError(t, err)
}

func waitForRestart(t *testing.T, tester commplatform.BotDriver, userID, channel, clusterName string) {
	t.Log("Waiting for restart...")
	originalTimeout := tester.Timeout()
	tester.SetTimeout(90 * time.Second)
	// 2, since time to time latest message becomes upgrade message right after begin message
	err := tester.WaitForMessagePosted(userID, channel, 2, func(content string) (bool, int, string) {
		return content == fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", clusterName), 0, ""
	})
	tester.SetTimeout(originalTimeout)
	require.NoError(t, err)
}

func hasAllColumns(msg string, headerColumnNames ...string) bool {
	for _, cn := range headerColumnNames {
		if !strings.Contains(msg, cn) {
			return false
		}
	}
	return true
}
func trimRightWhitespace(input string) string {
	lines := strings.Split(input, "\n")

	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, func(r rune) bool {
			return unicode.IsSpace(r)
		})
	}

	return strings.Join(lines, "\n")
}

func waitForLastPlaintextMessageEqual(driver commplatform.BotDriver, channelID, expectedMsg string) error {
	switch driver.Type() {
	case commplatform.TeamsBot:
		// in this case of a plain text message, Teams renderer uses Adaptive Cards format
		return driver.WaitForLastInteractiveMessagePostedEqual(driver.BotUserID(), channelID, interactive.CoreMessage{
			Message: api.Message{
				BaseBody: api.Body{
					Plaintext: expectedMsg,
				},
			},
		})
	default:
		return driver.WaitForLastMessageEqual(driver.BotUserID(), channelID, expectedMsg)
	}
}

func waitForLastPlaintextMessageWithHeaderEqual(cfg Config, driver commplatform.BotDriver, cmd, expectedBody string) error {
	return waitForLastMessageWithHeaderEqual(cfg, driver, cmd, expectedBody, false)
}

func waitForLastCodeBlockMessageWithHeaderEqual(cfg Config, driver commplatform.BotDriver, cmd, expectedBody string) error {
	return waitForLastMessageWithHeaderEqual(cfg, driver, cmd, expectedBody, true)
}

func waitForLastMessageWithHeaderEqual(cfg Config, driver commplatform.BotDriver, cmd, expectedBody string, asCodeBlock bool) error {
	cmdHeader := func(command string) string {
		return fmt.Sprintf("`%s` on `%s`", command, cfg.ClusterName)
	}

	switch driver.Type() {
	case commplatform.TeamsBot:
		// Teams renderer uses Adaptive Cards format to render header in a more readable way
		msg := interactive.CoreMessage{
			Description: cmdHeader(cmd),
			Message:     api.Message{},
		}
		if asCodeBlock {
			msg.Message.BaseBody.CodeBlock = expectedBody
		} else {
			msg.Message.BaseBody.Plaintext = expectedBody
		}

		return driver.WaitForLastInteractiveMessagePostedEqual(driver.BotUserID(), driver.Channel().ID(), msg)
	default:
		if asCodeBlock {
			expectedBody = codeBlock(expectedBody)
		}
		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(cmd), expectedBody)
		return driver.WaitForLastMessageEqual(driver.BotUserID(), driver.Channel().ID(), expectedMessage)
	}
}
