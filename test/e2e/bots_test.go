//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/filterengine/filters"
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
			DiscordEnabledName            string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_ENABLED"`
			DefaultDiscordChannelIDName   string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_CHANNELS_DEFAULT_ID"`
			SecondaryDiscordChannelIDName string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_DISCORD_CHANNELS_SECONDARY_ID"`
			BotkubePluginRepoURL          string `envconfig:"default=BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL"`
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

type AttachmentStatus = map[config.Level]string

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
	slackAnnotation     = "<http://botkube.io/*|botkube.io/*>"
	discordAnnotation   = "botkube.io/*"
	discordInvalidCmd   = "You must specify the type of resource to get. Use \"kubectl api-resources\" for a complete list of supported resources.\n\nerror: Required resource not specified.\nUse \"kubectl explain <resource>\" for a detailed description of that resource (e.g. kubectl explain pods).\nSee 'kubectl get -h' for help and examples\nexit status 1"
)

var (
	slackInvalidCmd = heredoc.Doc(`
				You must specify the type of resource to get. Use "kubectl api-resources" for a complete list of supported resources.

				error: Required resource not specified.
				Use "kubectl explain &lt;resource&gt;" for a detailed description of that resource (e.g. kubectl explain pods).
				See 'kubectl get -h' for help and examples
				exit status 1`)
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
		slackAnnotation,
		slackInvalidCmd,
		appCfg.Deployment.Envs.DefaultSlackChannelIDName,
		appCfg.Deployment.Envs.SecondarySlackChannelIDName,
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
		discordAnnotation,
		discordInvalidCmd,
		appCfg.Deployment.Envs.DefaultDiscordChannelIDName,
		appCfg.Deployment.Envs.SecondaryDiscordChannelIDName,
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
	annotation,
	invalidCmdTemplate,
	deployEnvChannelIDName,
	deployEnvSecondaryChannelIDName string,
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
	err = botDriver.WaitForInteractiveMessagePostedRecentlyEqual(botDriver.BotUserID(),
		botDriver.Channel().ID(),
		interactive.NewHelpMessage(config.CommPlatformIntegration(botDriver.Type()), appCfg.ClusterName, botDriver.BotName(), []string{"botkube/helm"}).Build(),
	)
	require.NoError(t, err)

	t.Log("Waiting for Bot message in channel...")
	err = botDriver.WaitForMessagePostedRecentlyEqual(botDriver.BotUserID(), botDriver.Channel().ID(), fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName))
	require.NoError(t, err)

	t.Log("Running actual test cases")

	t.Run("Ping", func(t *testing.T) {
		command := "ping"
		expectedMessage := fmt.Sprintf("`ping` on `%s`\n```\npong", appCfg.ClusterName)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Help", func(t *testing.T) {
		command := "help"
		expectedMessage := interactive.NewHelpMessage(config.CommPlatformIntegration(botDriver.Type()), appCfg.ClusterName, botDriver.BotName(), []string{"botkube/helm"}).Build()

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastInteractiveMessagePostedEqual(botDriver.BotUserID(),
			botDriver.Channel().ID(),
			expectedMessage,
		)

		assert.NoError(t, err)
	})

	t.Run("Filters list", func(t *testing.T) {
		command := "list filters"
		expectedBody := codeBlock(heredoc.Docf(`
			FILTER                  ENABLED DESCRIPTION
			NodeEventsChecker       false   Sends notifications on node level critical events.
			ObjectAnnotationChecker true    Filters or reroutes events based on %s Kubernetes resource annotations.`, annotation))
		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	// Those are a temporary tests. When we will extract kubectl and kubernetes as plugins
	// they won't be needed anymore.
	t.Run("Botkube Plugins", func(t *testing.T) {
		t.Run("Echo Executor success", func(t *testing.T) {
			command := "echo test"
			expectedBody := codeBlock(strings.ToUpper(command))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
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
			expectedMessage := ".... empty response _*&lt;cricket sounds&gt;*_ :cricket: :cricket: :cricket:"
			if botDriver.Type() == DiscordBot {
				expectedMessage = ".... empty response _*<cricket sounds>*_ :cricket: :cricket: :cricket:"
			}

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
			expectedMessage := codeBlock(heredoc.Doc(`
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
			command := fmt.Sprintf("kc get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Deployment with matching filter", func(t *testing.T) {
			command := fmt.Sprintf(`kc get deploy -n %s %s --filter='botkube'`, appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
					strings.Contains(msg, "botkube"), 0, ""
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("kc get configmap -n %s", appCfg.Deployment.Namespace)
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
			command := fmt.Sprintf(`kc get configmap -n %s --filter="unknown-thing"`, appCfg.Deployment.Namespace)
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
			command := fmt.Sprintf("kc get configmap %s -o yaml -n %s", globalConfigMapName, appCfg.Deployment.Namespace)
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
			command := "kc get role"
			expectedBody := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'role' resources in the 'default' Namespace on cluster '%s'. Use 'list executors' to see allowed executors.", appCfg.ClusterName))
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
			command := "kc get"
			expectedBody := codeBlock(invalidCmdTemplate)
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "kc get po --namespace team-b"
			expectedBody := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'po' resources in the 'team-b' Namespace on cluster '%s'. Use 'list executors' to see allowed executors.", appCfg.ClusterName))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Based on other bindings", func(t *testing.T) {
			t.Run("Wait for Deployment (the 2st binding)", func(t *testing.T) {
				command := fmt.Sprintf("kc wait deployment -n %s %s --for condition=Available=True", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
				assertionFn := func(msg string) (bool, int, string) {
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
						strings.Contains(msg, "deployment.apps/botkube condition met"), 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})

			t.Run("Exec (the 3rd binding which is disabled)", func(t *testing.T) {
				command := "kc exec"
				expectedBody := codeBlock(fmt.Sprintf("Sorry, the kubectl 'exec' command cannot be executed in the 'default' Namespace on cluster '%s'. Use 'list executors' to see allowed executors.", appCfg.ClusterName))
				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Pods (the 4th binding)", func(t *testing.T) {
				command := "kc get pods -A"
				expectedBody := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'pods' resources for all Namespaces on cluster '%s'. Use 'list executors' to see allowed executors.", appCfg.ClusterName))
				expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments (the 4th binding)", func(t *testing.T) {
				command := "kc get deploy -A"
				assertionFn := func(msg string) (bool, int, string) {
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
						strings.Contains(msg, "local-path-provisioner") &&
						strings.Contains(msg, "coredns") &&
						strings.Contains(msg, "botkube"), 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})
		})

		k8sPrefixTests := []string{"kubectl", "kc", "k"}
		for _, prefix := range k8sPrefixTests {
			t.Run(fmt.Sprintf("Get Pods with k8s prefix %s", prefix), func(t *testing.T) {
				command := fmt.Sprintf("%s get pods --namespace %s", prefix, appCfg.Deployment.Namespace)
				assertionFn := func(msg string) (bool, int, string) {
					headerColumnNames := []string{"NAME", "READY", "STATUS", "RESTART", "AGE"}
					containAllColumn := true
					for _, cn := range headerColumnNames {
						if !strings.Contains(msg, cn) {
							containAllColumn = false
						}
					}
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("`%s` on `%s`", command, appCfg.ClusterName))) &&
						containAllColumn, 0, ""
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
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
		var channelIDs []string
		for _, channel := range channels {
			channelIDs = append(channelIDs, channel.ID())
		}

		t.Log("Creating ConfigMap...")
		var cfgMapAlreadyDeleted bool
		cfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      botDriver.Channel().Name(),
				Namespace: appCfg.Deployment.Namespace,
				Labels:    configMapLabels,
			},
		}
		cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })

		t.Log("Expecting bot message in first channel...")
		attachAssertionFn := func(title, color, msg string) (bool, int, string) {
			expectedMsg := fmt.Sprintf("ConfigMap *%s/%s* has been created in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
			equal := title == "v1/configmaps created" && msg == expectedMsg && color == botDriver.GetColorByLevel(config.Info)
			if msg != expectedMsg {
				count := countMatchBlock(expectedMsg, msg)
				msgDiff := diff(expectedMsg, msg)
				return false, count, msgDiff
			}
			return equal, 0, ""
		}
		err = botDriver.WaitForLastMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), attachAssertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new in second channel...")

		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Updating ConfigMap...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message in all channels...")
		attachAssertionFn = func(title, _, msg string) (bool, int, string) {
			expectedMsg := fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
			equal := title == "v1/configmaps updated" && msg == expectedMsg
			if msg != expectedMsg {
				count := countMatchBlock(expectedMsg, msg)
				msgDiff := diff(expectedMsg, msg)
				return false, count, msgDiff
			}
			return equal, 0, ""
		}
		err = botDriver.WaitForMessagesPostedOnChannelsWithAttachment(botDriver.BotUserID(), channelIDs, attachAssertionFn)
		require.NoError(t, err)

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

		t.Log("Expecting bot message in second channel...")
		attachAssertionFn = func(title, _, msg string) (bool, int, string) {
			expectedMsg := fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
			equal := title == "v1/configmaps updated" && msg == expectedMsg
			if msg != expectedMsg {
				count := countMatchBlock(expectedMsg, msg)
				msgDiff := diff(expectedMsg, msg)
				return false, count, msgDiff
			}
			return equal, 0, ""
		}
		err = botDriver.WaitForLastMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), attachAssertionFn)

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
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		t.Log("Expecting bot message on first channel...")
		attachAssertionFn = func(title, _, msg string) (bool, int, string) {
			expectedMsg := fmt.Sprintf("ConfigMap *%s/%s* has been deleted in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
			equal := title == "v1/configmaps deleted" && msg == expectedMsg
			if msg != expectedMsg {
				count := countMatchBlock(expectedMsg, msg)
				msgDiff := diff(expectedMsg, msg)
				return false, count, msgDiff
			}
			return equal, 0, ""
		}
		err = botDriver.WaitForLastMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), attachAssertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't post anything new in second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		attachAssertionFn = func(title, _, msg string) (bool, int, string) {
			expectedMsg := fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
			equal := title == "v1/configmaps updated" && msg == expectedMsg
			if msg != expectedMsg {
				count := countMatchBlock(expectedMsg, msg)
				msgDiff := diff(expectedMsg, msg)
				return false, count, msgDiff
			}
			return equal, 0, ""
		}
		err = botDriver.WaitForLastMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), attachAssertionFn)
		require.NoError(t, err)
	})

	t.Run("Recommendations and actions", func(t *testing.T) {
		podCli := k8sCli.CoreV1().Pods(appCfg.Deployment.Namespace)

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
		pod, err = podCli.Create(context.Background(), pod, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedPod(t, podCli, pod.Name) })

		t.Log("Expecting bot event message...")
		assertionFn := func(title, color, msg string) (bool, int, string) {
			return title == "v1/pods created" &&
				strings.Contains(msg, "Recommendations:") &&
				strings.Contains(msg, fmt.Sprintf("- Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name)) &&
				strings.Contains(msg, fmt.Sprintf("- The 'latest' tag used in '%s' image of Pod '%s/%s' container '%s' should be avoided.", pod.Spec.Containers[0].Image, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)) &&
				color == botDriver.GetColorByLevel(config.Info), 0, ""
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), 2, assertionFn)
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
	})

	t.Run("List actions", func(t *testing.T) {
		command := "list actions"
		expectedBody := codeBlock(heredoc.Doc(`
			ACTION                    ENABLED  DISPLAY NAME
			describe-created-resource false    Describe created resource
			get-created-resource      true     Get created resource
			show-logs-on-error        false    Show logs on error`))

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("List executors", func(t *testing.T) {
		command := "list executors"
		expectedBody := codeBlock(heredoc.Doc(`
			EXECUTOR                  ENABLED
			botkube/echo@v1.0.1-devel true
			botkube/helm              true
			kubectl-allow-all         true
			kubectl-exec-cmd          false
			kubectl-read-only         true
			kubectl-wait-cmd          true`))

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("List sources", func(t *testing.T) {
		command := "list sources"
		expectedBody := codeBlock(heredoc.Doc(`
		SOURCE ENABLED DISPLAY NAME`))
		if botDriver.Type() == DiscordBot {
			expectedBody = codeBlock(heredoc.Doc(`
			SOURCE                  ENABLED DISPLAY NAME
			botkube/cm-watcher      true    K8s ConfigMaps changes
			k8s-annotated-cm-delete true    K8s ConfigMap delete events
			k8s-events              true    K8s recommendations
			k8s-pod-create-events   true`))
		}

		expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
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
