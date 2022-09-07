//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/pkg/filterengine/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
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
		}
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
	MessageWaitTimeout       time.Duration `envconfig:"default=10s"`
}

type DiscordConfig struct {
	BotName                  string `envconfig:"default=botkube"`
	TesterName               string `envconfig:"default=tester"`
	AdditionalContextMessage string `envconfig:"optional"`
	GuildID                  string
	TesterAppToken           string
	MessageWaitTimeout       time.Duration `envconfig:"default=10s"`
}

const (
	channelNamePrefix       = "test"
	welcomeText             = "Let the tests begin 🤞"
	pollInterval            = time.Second
	slackAnnotation         = "<http://botkube.io/*|botkube.io/*>"
	discordAnnotation       = "botkube.io/*"
	slackInvalidCmdTemplate = `Cluster: %s
				You must specify the type of resource to get. Use "kubectl api-resources" for a complete list of supported resources.

				error: Required resource not specified.
				Use "kubectl explain &lt;resource&gt;" for a detailed description of that resource (e.g. kubectl explain pods).
				See 'kubectl get -h' for help and examples
				exit status 1`
	discordInvalidCmdTemplate = "Cluster: %s\nYou must specify the type of resource to get. Use \"kubectl api-resources\" for a complete list of supported resources.\n\nerror: Required resource not specified.\nUse \"kubectl explain <resource>\" for a detailed description of that resource (e.g. kubectl explain pods).\nSee 'kubectl get -h' for help and examples\nexit status 1"
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
		slackInvalidCmdTemplate,
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
		discordInvalidCmdTemplate,
		appCfg.Deployment.Envs.DefaultDiscordChannelIDName,
		appCfg.Deployment.Envs.SecondaryDiscordChannelIDName,
	)
}

func newBotDriver(cfg Config, driverType BotType) (BotDriver, error) {
	switch driverType {
	case SlackBot:
		return newSlackTester(cfg.Slack)
	case DiscordBot:
		return newDiscordTester(cfg.Discord)
	}
	return nil, nil
}

func runBotTest(t *testing.T,
	appCfg Config,
	driverType BotType,
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
	revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, botDriver.Type(), channels)
	t.Cleanup(func() { revertDeployFn(t) })

	t.Log("Waiting for Deployment")
	err = waitForDeploymentReady(deployNsCli, appCfg.Deployment.Name, appCfg.Deployment.WaitTimeout)
	require.NoError(t, err)

	t.Log("Waiting for Bot message on channel from user")
	err = botDriver.WaitForMessagePostedRecentlyEqual(botDriver.BotUserID(), botDriver.Channel().ID(), fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName))
	require.NoError(t, err)

	t.Log("Running actual test cases")

	t.Run("Ping", func(t *testing.T) {
		command := "ping"
		expectedMessage := fmt.Sprintf("pong from cluster '%s'", appCfg.ClusterName)

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageContains(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Filters list", func(t *testing.T) {
		command := "filters list"
		expectedMessage := codeBlock(heredoc.Doc(fmt.Sprintf(`
			FILTER                  ENABLED DESCRIPTION
			NodeEventsChecker       true    Sends notifications on node level critical events.
			ObjectAnnotationChecker true    Checks if annotations %s present in object specs and filters them.`, annotation)))

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Commands list", func(t *testing.T) {
		command := "commands list"
		expectedMessage := codeBlock(heredoc.Doc(`
			Enabled executors:
			  kubectl:
			    kubectl-allow-all:
			      namespaces:
			        include:
			          - .*
			      enabled: true
			      commands:
			        verbs:
			          - get
			        resources:
			          - deployments
			    kubectl-read-only:
			      namespaces:
			        include:
			          - botkube
			          - default
			      enabled: true
			      commands:
			        verbs:
			          - api-resources
			          - api-versions
			          - cluster-info
			          - describe
			          - diff
			          - explain
			          - get
			          - logs
			          - top
			          - auth
			        resources:
			          - deployments
			          - pods
			          - namespaces
			          - daemonsets
			          - statefulsets
			          - storageclasses
			          - nodes
			          - configmaps
			      defaultNamespace: default
			      restrictAccess: false
			    kubectl-wait-cmd:
			      namespaces:
			        include:
			          - botkube
			          - default
			      enabled: true
			      commands:
			        verbs:
			          - wait
			        resources: []
			      restrictAccess: false`))

		t.Run("With default cluster", func(t *testing.T) {
			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err := botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With custom cluster name", func(t *testing.T) {
			command := fmt.Sprintf("commands list --cluster-name %s", appCfg.ClusterName)

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With unknown cluster name", func(t *testing.T) {
			command := "commands list --cluster-name non-existing"

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			t.Log("Ensuring bot didn't write anything new...")
			time.Sleep(appCfg.Slack.MessageWaitTimeout)
			// Same expected message as before
			err = botDriver.WaitForLastMessageContains(botDriver.TesterUserID(), botDriver.Channel().ID(), command)
			assert.NoError(t, err)
		})
	})

	t.Run("Executor", func(t *testing.T) {
		t.Run("Get Deployment", func(t *testing.T) {
			command := fmt.Sprintf("get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg string) bool {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg, "botkube")
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("get configmap -n %s", appCfg.Deployment.Namespace)
			assertionFn := func(msg string) bool {
				return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg, "kube-root-ca.crt") &&
					strings.Contains(msg, "botkube-global-config")
			}

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get forbidden resource", func(t *testing.T) {
			command := "get ingress"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'ingress' resources in the 'default' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify unknown command", func(t *testing.T) {
			command := "unknown"
			expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify invalid command", func(t *testing.T) {
			command := "get"
			expectedMessage := codeBlock(heredoc.Docf(invalidCmdTemplate, appCfg.ClusterName))

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "get po --namespace team-b"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'po' resources in the 'team-b' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
			err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Based on other bindings", func(t *testing.T) {
			t.Run("Wait for Deployment (the 2st binding)", func(t *testing.T) {
				command := fmt.Sprintf("wait deployment -n %s %s --for condition=Available=True", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
				assertionFn := func(msg string) bool {
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg, "deployment.apps/botkube condition met")
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})

			t.Run("Exec (the 3rd binding which is disabled)", func(t *testing.T) {
				command := "exec"
				expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Pods (the 4th binding)", func(t *testing.T) {
				command := "get pods -A"
				expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'pods' resources for all Namespaces on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments (the 4th binding)", func(t *testing.T) {
				command := "get deploy -A"
				assertionFn := func(msg string) bool {
					return strings.Contains(msg, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg, "local-path-provisioner") &&
						strings.Contains(msg, "coredns") &&
						strings.Contains(msg, "botkube")
				}

				botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
				err = botDriver.WaitForMessagePosted(botDriver.BotUserID(), botDriver.Channel().ID(), 1, assertionFn)
				assert.NoError(t, err)
			})
		})
	})

	t.Run("Multi-channel notifications", func(t *testing.T) {
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
			},
		}
		cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })

		t.Log("Expecting bot message in first channel...")
		attachAssertionFn := func(title, _, msg string) bool {
			return title == "v1/configmaps created" &&
				msg == fmt.Sprintf("ConfigMap *%s/%s* has been created in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), attachAssertionFn)
		require.NoError(t, err)

		t.Log("Expecting no bot message in second channel...")
		expectedMessage := fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName)
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
		attachAssertionFn = func(title, _, msg string) bool {
			return title == "v1/configmaps updated" &&
				msg == fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
		}
		err = botDriver.WaitForMessagesPostedOnChannelsWithAttachment(botDriver.BotUserID(), channelIDs, attachAssertionFn)
		require.NoError(t, err)

		t.Log("Stopping notifier...")
		command := "notifier stop"
		expectedMessage = codeBlock(fmt.Sprintf("Sure! I won't send you notifications from cluster '%s' here.", appCfg.ClusterName))

		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from second channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are enabled here.", appCfg.ClusterName))
		botDriver.PostMessageToBot(t, botDriver.SecondChannel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.SecondChannel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from first channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))
		botDriver.PostMessageToBot(t, botDriver.Channel().Identifier(), command)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		assert.NoError(t, err)

		t.Log("Updating ConfigMap once again...")
		cfgMap.Data = map[string]string{
			"operation": "update-second",
		}
		_, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Ensuring bot didn't write anything new on first channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		// Same expected message as before
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Expecting bot message on second channel...")
		attachAssertionFn = func(title, _, msg string) bool {
			return title == "v1/configmaps updated" &&
				msg == fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), attachAssertionFn)

		t.Log("Starting notifier")
		command = "notifier start"
		expectedMessage = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
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

		t.Log("Ensuring bot didn't write anything new...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = botDriver.WaitForLastMessageEqual(botDriver.BotUserID(), botDriver.Channel().ID(), expectedMessage)
		require.NoError(t, err)

		t.Log("Deleting ConfigMap")
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		t.Log("Expecting bot message on first channel...")
		attachAssertionFn = func(title, _, msg string) bool {
			return title == "v1/configmaps deleted" &&
				msg == fmt.Sprintf("ConfigMap *%s/%s* has been deleted in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), attachAssertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't write anything new on second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		attachAssertionFn = func(title, _, msg string) bool {
			return title == "v1/configmaps updated" &&
				msg == fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName)
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.SecondChannel().ID(), attachAssertionFn)
		require.NoError(t, err)
	})

	t.Run("Recommendations", func(t *testing.T) {
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

		t.Log("Expecting bot message...")
		assertionFn := func(title, color, msg string) bool {
			return title == "v1/pods created" &&
				strings.Contains(msg, "Recommendations:") &&
				strings.Contains(msg, fmt.Sprintf("- Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name)) &&
				strings.Contains(msg, fmt.Sprintf("- The 'latest' tag used in '%s' image of Pod '%s/%s' container '%s' should be avoided.", pod.Spec.Containers[0].Image, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name))
		}
		err = botDriver.WaitForMessagePostedWithAttachment(botDriver.BotUserID(), botDriver.Channel().ID(), assertionFn)
		require.NoError(t, err)
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
