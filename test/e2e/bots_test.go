//go:build integration

package e2e

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	v13 "k8s.io/api/networking/v1"
	v12 "k8s.io/api/rbac/v1"
	v14 "k8s.io/client-go/kubernetes/typed/networking/v1"
	v15 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/internal/source/kubernetes/filterengine/filters"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/test/fake"
)

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
			BotkubePluginRepoURL          string `envconfig:"default=BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL"`
			LabelActionEnabledName        string `envconfig:"default=BOTKUBE_ACTIONS_LABEL-CREATED-SVC-RESOURCE_ENABLED"`
			StandaloneActionEnabledName   string `envconfig:"default=BOTKUBE_ACTIONS_GET-CREATED-RESOURCE_ENABLED"`
		}
	}
	Plugins   fake.PluginConfig
	ConfigMap struct {
		Namespace string `envconfig:"default=botkube"`
	}
	ClusterName string `envconfig:"default=sample"`
	Slack       SlackConfig
	Discord     DiscordConfig
}

type SlackConfig struct {
	BotName                  string `envconfig:"default=botkube"`
	TesterName               string `envconfig:"default=tester"`
	AdditionalContextMessage string `envconfig:"optional"`
	TesterAppToken           string
	MessageWaitTimeout       time.Duration `envconfig:"default=30s"`
}

type DiscordConfig struct {
	BotName                  string `envconfig:"optional"`
	BotID                    string `envconfig:"default=983294404108378154"`
	TesterName               string `envconfig:"optional"`
	TesterID                 string `envconfig:"default=1020384322114572381"`
	AdditionalContextMessage string `envconfig:"optional"`
	GuildID                  string
	TesterAppToken           string
	MessageWaitTimeout       time.Duration `envconfig:"default=30s"`
}

const (
	channelNamePrefix   = "test"
	welcomeText         = "Let the tests begin 🤞"
	pollInterval        = time.Second
	globalConfigMapName = "botkube-global-config"
)

var (
	discordInvalidCmd = heredoc.Doc(`
				You must specify the type of resource to get. Use "kubectl api-resources" for a complete list of supported resources.

				error: Required resource not specified.
				Use "kubectl explain <resource>" for a detailed description of that resource (e.g. kubectl explain pods).
				See 'kubectl get -h' for help and examples

				exit status 1`)
	slackInvalidCmd = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(discordInvalidCmd)
	configMapLabels = map[string]string{
		"test.botkube.io": "true",
	}
)

func TestSlack(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	runBotTest(t,
		appCfg,
		SlackBot,
		slackInvalidCmd,
		appCfg.Deployment.Envs.DefaultSlackChannelIDName,
		appCfg.Deployment.Envs.SecondarySlackChannelIDName,
		appCfg.Deployment.Envs.ThirdSlackChannelIDName,
	)
}

func TestDiscord(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	runBotTest(t,
		appCfg,
		DiscordBot,
		discordInvalidCmd,
		appCfg.Deployment.Envs.DefaultDiscordChannelIDName,
		appCfg.Deployment.Envs.SecondaryDiscordChannelIDName,
		appCfg.Deployment.Envs.ThirdDiscordChannelIDName,
	)
}

func newBotDriver(cfg Config, driverType DriverType) (BotDriver, error) {
	switch driverType {
	case SlackBot:
		return newSlackDriver(cfg.Slack)
	case DiscordBot:
		return newDiscordDriver(cfg.Discord)
	}
	return nil, nil
}

func runBotTest(t *testing.T,
	appCfg Config,
	driverType DriverType,
	invalidCmdTemplate,
	deployEnvChannelIDName,
	deployEnvSecondaryChannelIDName,
	deployEnvRbacChannelIDName string,
) {
	t.Logf("Creating API client with provided token for %s...", driverType)
	botDriver, err := newBotDriver(appCfg, driverType)
	require.NoError(t, err)

	t.Log("Creating K8s client...")
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", appCfg.KubeconfigPath)
	require.NoError(t, err)
	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(t, err)

	t.Log("Starting plugin server...")
	indexEndpoint, startServerFn := fake.NewPluginServer(appCfg.Plugins)
	go func() {
		require.NoError(t, startServerFn())
	}()

	t.Logf("Setting up test %s setup...", driverType)
	botDriver.InitUsers(t)

	cleanUpFns := botDriver.InitChannels(t)
	for _, fn := range cleanUpFns {
		t.Cleanup(fn)
	}

	channels := map[string]Channel{
		deployEnvChannelIDName:          botDriver.Channel(),
		deployEnvSecondaryChannelIDName: botDriver.SecondChannel(),
		deployEnvRbacChannelIDName:      botDriver.ThirdChannel(),
	}

	for _, currentChannel := range channels {
		botDriver.PostInitialMessage(t, currentChannel.Identifier())
		botDriver.InviteBotToChannel(t, currentChannel.ID())
	}

	t.Log("Patching Deployment with test env variables...")
	deployNsCli := k8sCli.AppsV1().Deployments(appCfg.Deployment.Namespace)
	revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, botDriver.Type(), channels, indexEndpoint)
	t.Cleanup(func() { revertDeployFn(t) })

	t.Log("Waiting for Deployment")
	err = waitForDeploymentReady(deployNsCli, appCfg.Deployment.Name, appCfg.Deployment.WaitTimeout)
	require.NoError(t, err)

	cmdHeader := func(command string) string {
		return fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName)
	}

	// TODO: configure and use MessageWaitTimeout from an (app) Config as it targets both Slack and Discord.
	// Discord bot needs a bit more time to connect to Discord API.
	time.Sleep(appCfg.Discord.MessageWaitTimeout)
	t.Log("Waiting for interactive help")
	expMessage := interactive.NewHelpMessage(config.CommPlatformIntegration(botDriver.Type()), appCfg.ClusterName, []string{"botkube/helm", "botkube/kubectl"}).Build()
	expMessage.ReplaceBotNamePlaceholder(botDriver.BotName())
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
		expectedMessage.ReplaceBotNamePlaceholder(botDriver.BotName())
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastInteractiveMessagePostedEqual(botDriver.BotUserID(),
			botDriver.Channel().ID(),
			expectedMessage,
		)

		assert.NoError(t, err)
	})

	// Those are a temporary tests. When we will extract kubectl and kubernetes as plugins
	// they won't be needed anymore.
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
			expectedBody := ".... empty response _*&lt;cricket sounds&gt;*_ :cricket: :cricket: :cricket:"
			if botDriver.Type() == DiscordBot {
				expectedBody = ".... empty response _*<cricket sounds>*_ :cricket: :cricket: :cricket:"
			}
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
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
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("ConfigMap watcher source", func(t *testing.T) {
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
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMsg)
			require.NoError(t, err)
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
			t.Log("Ensuring bot didn't post anything new...")
			time.Sleep(appCfg.Slack.MessageWaitTimeout)
			// Same expected message as before
			err = botDriver.WaitForLastMessageContains(botDriver.TesterUserID(), botDriver.Channel().ID(), command)
			assert.NoError(t, err)
		})
	})

	t.Run("Executor", func(t *testing.T) {
		t.Run("Get Deployment", func(t *testing.T) {
			command := fmt.Sprintf("kubectl get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Deployment with matching filter", func(t *testing.T) {
			command := fmt.Sprintf(`kubectl get deploy -n %s %s --filter='botkube'`, appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("kubectl get configmap -n %s", appCfg.Deployment.Namespace)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "kube-root-ca.crt") &&
					strings.Contains(msg, "botkube-global-config"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap with mismatching filter", func(t *testing.T) {
			command := fmt.Sprintf(`kubectl get configmap -n %s --filter="unknown-thing"`, appCfg.Deployment.Namespace)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					!strings.Contains(msg, "kube-root-ca.crt") &&
					!strings.Contains(msg, "botkube-global-config"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Receive large output as plaintext file with executor command as message", func(t *testing.T) {
			command := fmt.Sprintf("kubectl get configmap %s -o yaml -n %s", globalConfigMapName, appCfg.Deployment.Namespace)
			fileUploadAssertionFn := func(title, mimetype string) bool {
				return title == "Response.txt" && strings.Contains(mimetype, "text/plain")
			}
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePostedWithFileUpload(botDriver.BotUserID(), botDriver.Channel().ID(), fileUploadAssertionFn)
			assert.NoError(t, err)

			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))), 0, ""
			}
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
		})

		t.Run("Get forbidden resource", func(t *testing.T) {
			command := "kubectl get role"
			expectedBody := codeBlock(heredoc.Docf(`
				Error from server (Forbidden): roles.rbac.authorization.k8s.io is forbidden: User "kubectl-first-channel" cannot list resource "roles" in API group "rbac.authorization.k8s.io" in the namespace "default"

				exit status 1`))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
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
			expectedBody := codeBlock(invalidCmdTemplate)
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "kubectl get po --namespace team-b"
			expectedBody := codeBlock(heredoc.Docf(`
				Error from server (Forbidden): pods is forbidden: User "kubectl-first-channel" cannot list resource "pods" in API group "" in the namespace "team-b"

				exit status 1`))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
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
				Defaulted container "botkube" out of: botkube, cfg-watcher
				Error from server (Forbidden): pods "botkube-pod" is forbidden: User "kubectl-first-channel" cannot create resource "pods/exec" in API group "" in the namespace "botkube"

				exit status 1`))
				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)

				podName, err := regexp.Compile(`"botkube-.*-.*" is`)
				assert.NoError(t, err)

				assertionFn := func(msg string) (bool, int, string) {
					msg = podName.ReplaceAllString(msg, `"botkube-pod" is`)
					msg = trimTrailingLine(msg)
					if !strings.EqualFold(expectedMessage, msg) {
						count := countMatchBlock(expectedMessage, msg)
						msgDiff := diff(expectedMessage, msg)
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

				expectedBody := codeBlock(heredoc.Docf(`
				Error from server (Forbidden): pods is forbidden: User "kubectl-first-channel" cannot list resource "pods" in API group "" at the cluster scope

				exit status 1`))

				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(expandedCommand), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCommand)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments with alias", func(t *testing.T) {
				aliasedCommand := "kgda"
				expandedCommand := "kubectl get deployments -A"
				assertionFn := func(msg string) (bool, int, string) {
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", expandedCommand, appCfg.ClusterName))) &&
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
					headerColumnNames := []string{"NAME", "READY", "STATUS", "RESTART", "AGE"}
					containAllColumn := true
					for _, cn := range headerColumnNames {
						if !strings.Contains(msg, cn) {
							containAllColumn = false
						}
					}
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", expandedCmd, appCfg.ClusterName))) &&
						containAllColumn, 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), aliasedCmd)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Multi-channel notifications", func(t *testing.T) {
		t.Log("Getting notifier status from second channel...")
		command := "status notifications"
		expectedBody := codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))
		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Starting notifier in second channel...")
		command = "enable notifications"
		expectedBody = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
		expectedMessage = fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		require.NoError(t, err)

		cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
		// Third (RBAC) channel is isolated from this
		channelIDs := []string{channels[deployEnvChannelIDName].ID(), channels[deployEnvSecondaryChannelIDName].ID()}

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
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, ExpAttachmentInput{
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

		t.Log("Ensuring bot didn't post anything new in second channel...")

		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Updating ConfigMap...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		updateCMEventTime := time.Now()
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message in all channels...")
		for _, channelID := range channelIDs {
			err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), channelID, 2, ExpAttachmentInput{
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
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
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
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		require.NoError(t, err)

		secondCMUpdate := ExpAttachmentInput{
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
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Creating and deleting ignored ConfigMap")
		ignoredCfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-ignored", botDriver.Channel().Name()),
				Namespace: appCfg.Deployment.Namespace,
				Annotations: map[string]string{
					filters.DisableAnnotation: "true",
				},
			},
		}
		_, err = cfgMapCli.Create(context.Background(), ignoredCfgMap, metav1.CreateOptions{})
		require.NoError(t, err)
		err = cfgMapCli.Delete(context.Background(), ignoredCfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Deleting ConfigMap")
		deleteCMEventTime := time.Now()
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		t.Log("Expecting bot message on first channel...")
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, ExpAttachmentInput{
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
		})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new in second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), 2, secondCMUpdate)
		require.NoError(t, err)
	})

	t.Run("Recommendations and actions", func(t *testing.T) {
		podCli := k8sCli.CoreV1().Pods(appCfg.Deployment.Namespace)
		svcCli := k8sCli.CoreV1().Services(appCfg.Deployment.Namespace)

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
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 3, ExpAttachmentInput{
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
		cmdHeaderWithAuthor := func(command, author string) string {
			return fmt.Sprintf("`%s` on `%s` by %s", command, appCfg.ClusterName, author)
		}
		command := fmt.Sprintf(`kubectl get pod -n %s %s`, pod.Namespace, pod.Name)
		automationAssertionFn := func(content string) (bool, int, string) {
			return strings.Contains(content, cmdHeaderWithAuthor(command, "Automation \"Get created resource\"")) &&
					strings.Contains(content, "NAME") && strings.Contains(content, "READY") && strings.Contains(content, "STATUS") && // command output header
					strings.Count(content, pod.Name) == 2, // command + 1 occurrence in the command output
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
		err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, automationAssertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot automation was executed and label created Service...")
		err = wait.PollImmediate(pollInterval, appCfg.Slack.MessageWaitTimeout, func() (done bool, err error) {
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
			EXECUTOR                  ENABLED ALIASES
			botkube/echo@v1.0.1-devel true    e
			botkube/helm              true    
			botkube/kubectl           true    k, kc`))

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

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("List sources", func(t *testing.T) {
		command := "list sources"
		expectedBody := codeBlock(heredoc.Doc(`
		SOURCE ENABLED`))
		if botDriver.Type() == DiscordBot {
			expectedBody = codeBlock(heredoc.Doc(`
			SOURCE             ENABLED
			botkube/cm-watcher true
			botkube/kubernetes true`))
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
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.ThirdChannel().ID(), expectedMsg)
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
			err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, ExpAttachmentInput{
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
			t.Log("Creating RBAC ConfigMap for ChannelName mapping...")
			clusterRole := &v12.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ing-rbac-channel",
				},
				Rules: []v12.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"ingresses"},
						Verbs:     []string{"get"},
					},
				},
			}

			t.Log("Creating RBAC ClusterRole for ChannelName mapping...")
			clusterRoleCli := k8sCli.RbacV1().ClusterRoles()
			cr, err := clusterRoleCli.Create(context.Background(), clusterRole, metav1.CreateOptions{})
			require.NoError(t, err)

			clusterRoleBinding := &v12.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ing-rbac-channel",
				},
				RoleRef: v12.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "ing-rbac-channel",
				},
				Subjects: []v12.Subject{
					{
						Kind:     "Group",
						Name:     botDriver.ThirdChannel().Identifier(),
						APIGroup: "rbac.authorization.k8s.io",
					},
				},
			}

			t.Log("Creating RBAC ClusterRoleBinding for ChannelName mapping...")
			clusterRoleBindingCli := k8sCli.RbacV1().ClusterRoleBindings()
			crb, err := clusterRoleBindingCli.Create(context.Background(), clusterRoleBinding, metav1.CreateOptions{})
			require.NoError(t, err)

			ing := &v13.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ing-rbac-channel",
				},
				Spec: v13.IngressSpec{
					DefaultBackend: &v13.IngressBackend{
						Service: &v13.IngressServiceBackend{
							Name: "test",
							Port: v13.ServiceBackendPort{
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
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.ThirdChannel().ID(), 1, assertionFn)
			assert.NoError(t, err)

			t.Cleanup(func() { cleanupCreatedIng(t, ingressCli, ingress.Name) })
			t.Cleanup(func() { cleanupCreatedClusterRole(t, clusterRoleCli, cr.Name) })
			t.Cleanup(func() { cleanupCreatedClusterRoleBinding(t, clusterRoleBindingCli, crb.Name) })
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
func cleanupCreatedIng(t *testing.T, ingressCli v14.IngressInterface, name string) {
	t.Log("Cleaning up created Ingress...")
	err := ingressCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedClusterRole(t *testing.T, clusterRoleCli v15.ClusterRoleInterface, name string) {
	t.Log("Cleaning up created ClusterRole...")
	err := clusterRoleCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func cleanupCreatedClusterRoleBinding(t *testing.T, clusterRoleBindingCli v15.ClusterRoleBindingInterface, name string) {
	t.Log("Cleaning up created ClusterRoleBinding...")
	err := clusterRoleBindingCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}
