//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/bwmarrin/discordgo"
	"github.com/kubeshop/botkube/pkg/filterengine/filters"
	"github.com/slack-go/slack"
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
	channelNamePrefix = "test"
	welcomeText       = "Let the tests begin 🤞"
	pollInterval      = time.Second
)

func TestSlack(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	t.Log("Creating Slack API client with provided token...")
	slackTester, err := newSlackTester(appCfg.Slack)
	require.NoError(t, err)

	t.Log("Creating K8s client...")
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", appCfg.KubeconfigPath)
	require.NoError(t, err)
	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(t, err)

	t.Log("Setting up test Slack setup...")
	botUserID := slackTester.FindUserIDForBot(t)
	testerUserID := slackTester.FindUserIDForTester(t)

	channel, cleanupChannelFn := slackTester.CreateChannel(t)
	t.Cleanup(func() { cleanupChannelFn(t) })
	secondChannel, cleanupSecondChannelFn := slackTester.CreateChannel(t)
	t.Cleanup(func() { cleanupSecondChannelFn(t) })

	channels := map[string]*slack.Channel{
		appCfg.Deployment.Envs.DefaultSlackChannelIDName:   channel,
		appCfg.Deployment.Envs.SecondarySlackChannelIDName: secondChannel,
	}
	for _, currentChannel := range channels {
		slackTester.PostInitialMessage(t, currentChannel.Name)
		slackTester.InviteBotToChannel(t, botUserID, currentChannel.ID)
	}

	t.Log("Patching Deployment with test env variables...")
	deployNsCli := k8sCli.AppsV1().Deployments(appCfg.Deployment.Namespace)
	revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, channels, nil)
	t.Cleanup(func() { revertDeployFn(t) })

	t.Log("Waiting for Deployment")
	err = waitForDeploymentReady(deployNsCli, appCfg.Deployment.Name, appCfg.Deployment.WaitTimeout)
	require.NoError(t, err)

	t.Log("Waiting for Bot message on channel...")
	err = slackTester.WaitForMessagePostedRecentlyEqual(botUserID, channel.ID, fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName))
	require.NoError(t, err)

	t.Log("Running actual test cases")

	t.Run("Ping", func(t *testing.T) {
		command := "ping"
		expectedMessage := fmt.Sprintf("pong from cluster '%s'", appCfg.ClusterName)

		slackTester.PostMessageToBot(t, channel.Name, command)
		err := slackTester.WaitForLastMessageContains(botUserID, channel.ID, expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Filters list", func(t *testing.T) {
		command := "filters list"
		expectedMessage := codeBlock(heredoc.Doc(`
			FILTER                  ENABLED DESCRIPTION
			NodeEventsChecker       true    Sends notifications on node level critical events.
			ObjectAnnotationChecker true    Checks if annotations <http://botkube.io/*|botkube.io/*> present in object specs and filters them.`))

		slackTester.PostMessageToBot(t, channel.Name, command)
		err := slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
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
			slackTester.PostMessageToBot(t, channel.Name, command)
			err := slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With custom cluster name", func(t *testing.T) {
			command := fmt.Sprintf("commands list --cluster-name %s", appCfg.ClusterName)

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With unknown cluster name", func(t *testing.T) {
			command := "commands list --cluster-name non-existing"

			slackTester.PostMessageToBot(t, channel.Name, command)
			t.Log("Ensuring bot didn't write anything new...")
			time.Sleep(appCfg.Slack.MessageWaitTimeout)
			// Same expected message as before
			err = slackTester.WaitForLastMessageContains(testerUserID, channel.ID, command)
			assert.NoError(t, err)
		})
	})

	t.Run("Executor", func(t *testing.T) {
		t.Run("Get Deployment", func(t *testing.T) {
			command := fmt.Sprintf("get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg slack.Message) bool {
				return strings.Contains(msg.Text, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg.Text, "botkube")
			}

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("get configmap -n %s", appCfg.Deployment.Namespace)
			assertionFn := func(msg slack.Message) bool {
				return strings.Contains(msg.Text, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg.Text, "kube-root-ca.crt") &&
					strings.Contains(msg.Text, "botkube-global-config")
			}

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get forbidden resource", func(t *testing.T) {
			command := "get ingress"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'ingress' resources in the 'default' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify unknown command", func(t *testing.T) {
			command := "unknown"
			expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify invalid command", func(t *testing.T) {
			command := "get"
			expectedMessage := codeBlock(heredoc.Docf(`Cluster: %s
				You must specify the type of resource to get. Use "kubectl api-resources" for a complete list of supported resources.

				error: Required resource not specified.
				Use "kubectl explain &lt;resource&gt;" for a detailed description of that resource (e.g. kubectl explain pods).
				See 'kubectl get -h' for help and examples
				exit status 1`, appCfg.ClusterName))

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "get po --namespace team-b"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'po' resources in the 'team-b' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			slackTester.PostMessageToBot(t, channel.Name, command)
			err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Based on other bindings", func(t *testing.T) {
			t.Run("Wait for Deployment (the 2st binding)", func(t *testing.T) {
				command := fmt.Sprintf("wait deployment -n %s %s --for condition=Available=True", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
				assertionFn := func(msg slack.Message) bool {
					return strings.Contains(msg.Text, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg.Text, "deployment.apps/botkube condition met")
				}

				slackTester.PostMessageToBot(t, channel.Name, command)
				err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
				assert.NoError(t, err)
			})

			t.Run("Exec (the 3rd binding which is disabled)", func(t *testing.T) {
				command := "exec"
				expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

				slackTester.PostMessageToBot(t, channel.Name, command)
				err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Pods (the 4th binding)", func(t *testing.T) {
				command := "get pods -A"
				expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'pods' resources for all Namespaces on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

				slackTester.PostMessageToBot(t, channel.Name, command)
				err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments (the 4th binding)", func(t *testing.T) {
				command := "get deploy -A"
				assertionFn := func(msg slack.Message) bool {
					return strings.Contains(msg.Text, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg.Text, "local-path-provisioner") &&
						strings.Contains(msg.Text, "coredns") &&
						strings.Contains(msg.Text, "botkube")
				}

				slackTester.PostMessageToBot(t, channel.Name, command)
				err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
				assert.NoError(t, err)
			})
		})
	})

	t.Run("Multi-channel notifications", func(t *testing.T) {
		cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
		var channelIDs []string
		for _, channel := range channels {
			channelIDs = append(channelIDs, channel.ID)
		}

		t.Log("Creating ConfigMap...")
		var cfgMapAlreadyDeleted bool
		cfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      channel.Name,
				Namespace: appCfg.Deployment.Namespace,
			},
		}
		cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })

		t.Log("Expecting bot message in first channel...")
		assertionFn := func(msg slack.Message) bool {
			return doesSlackMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps created",
				"2eb886",
				fmt.Sprintf("ConfigMap *%s/%s* has been created in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Expecting no bot message in second channel...")
		expectedMessage := fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName)
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		err = slackTester.WaitForLastMessageEqual(botUserID, secondChannel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Updating ConfigMap...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message in all channels...")
		assertionFn = func(msg slack.Message) bool {
			return doesSlackMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps updated",
				"daa038",
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagesPostedOnChannels(botUserID, channelIDs, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Stopping notifier...")
		command := "notifier stop"
		expectedMessage = codeBlock(fmt.Sprintf("Sure! I won't send you notifications from cluster '%s' here.", appCfg.ClusterName))

		slackTester.PostMessageToBot(t, channel.Name, command)
		err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from second channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are enabled here.", appCfg.ClusterName))
		slackTester.PostMessageToBot(t, secondChannel.Name, command)
		err = slackTester.WaitForLastMessageEqual(botUserID, secondChannel.ID, expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from first channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))
		slackTester.PostMessageToBot(t, channel.Name, command)
		err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
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
		err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Expecting bot message on second channel...")
		assertionFn = func(msg slack.Message) bool {
			return doesSlackMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps updated",
				"daa038",
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagePosted(botUserID, secondChannel.ID, 1, assertionFn)

		t.Log("Starting notifier")
		command = "notifier start"
		expectedMessage = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
		slackTester.PostMessageToBot(t, channel.Name, command)
		err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Creating and deleting ignored ConfigMap")
		ignoredCfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-ignored", channel.Name),
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
		err = slackTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Deleting ConfigMap")
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		t.Log("Expecting bot message on first channel...")
		assertionFn = func(msg slack.Message) bool {
			return doesSlackMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps deleted",
				"a30200",
				fmt.Sprintf("ConfigMap *%s/%s* has been deleted in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't write anything new on second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		assertionFn = func(msg slack.Message) bool {
			return doesSlackMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps updated",
				"daa038",
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagePosted(botUserID, secondChannel.ID, 1, assertionFn)
		require.NoError(t, err)
	})

	t.Run("Recommendations", func(t *testing.T) {
		podCli := k8sCli.CoreV1().Pods(appCfg.Deployment.Namespace)

		t.Log("Creating Pod...")
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      channel.Name,
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
		assertionFn := func(msg slack.Message) bool {
			if len(msg.Attachments) != 1 {
				return false
			}

			attachment := msg.Attachments[0]
			title := attachment.Title

			if len(attachment.Fields) != 1 {
				return false
			}

			fieldMessage := attachment.Fields[0].Value
			return title == "v1/pods created" &&
				strings.Contains(fieldMessage, "Recommendations:") &&
				strings.Contains(fieldMessage, fmt.Sprintf("- Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name)) &&
				strings.Contains(fieldMessage, fmt.Sprintf("- The 'latest' tag used in '%s' image of Pod '%s/%s' container '%s' should be avoided.", pod.Spec.Containers[0].Image, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name))
		}
		err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)
	})
}

func TestDiscord(t *testing.T) {
	t.Log("Loading configuration...")
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err)

	t.Log("Creating Discord API client with provided token...")
	discordTester, err := newDiscordTester(appCfg.Discord)
	require.NoError(t, err)

	t.Log("Creating K8s client...")
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", appCfg.KubeconfigPath)
	require.NoError(t, err)
	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(t, err)

	t.Log("Setting up test Slack setup...")
	botUserID := discordTester.FindUserIDForBot(t)
	testerUserID := discordTester.FindUserIDForTester(t)
	t.Logf("Just loaded botUserID...: %+v", botUserID)

	channel, cleanupChannelFn := discordTester.CreateChannel(t)
	t.Cleanup(func() { cleanupChannelFn(t) })
	secondChannel, cleanupSecondChannelFn := discordTester.CreateChannel(t)
	t.Cleanup(func() { cleanupSecondChannelFn(t) })

	channels := map[string]*discordgo.Channel{
		appCfg.Deployment.Envs.DefaultDiscordChannelIDName:   channel,
		appCfg.Deployment.Envs.SecondaryDiscordChannelIDName: secondChannel,
	}

	for _, currentChannel := range channels {
		discordTester.PostInitialMessage(t, currentChannel.ID)
		discordTester.InviteBotToChannel(t, botUserID, currentChannel.ID)
	}

	t.Log("Patching Deployment with test env variables...")
	deployNsCli := k8sCli.AppsV1().Deployments(appCfg.Deployment.Namespace)
	revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, nil, channels)
	t.Cleanup(func() { revertDeployFn(t) })

	t.Log("Waiting for Deployment")
	err = waitForDeploymentReady(deployNsCli, appCfg.Deployment.Name, appCfg.Deployment.WaitTimeout)
	require.NoError(t, err)

	t.Log("Waiting for Bot message on channel from user")
	err = discordTester.WaitForMessagePostedRecentlyEqual(botUserID, channel.ID, fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName))
	require.NoError(t, err)

	t.Log("Running actual test cases")

	t.Run("Ping", func(t *testing.T) {
		command := "ping"
		expectedMessage := fmt.Sprintf("pong from cluster '%s'", appCfg.ClusterName)

		//discordTester.PostMessageToBot(t, channel.ID, command)
		discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
		err := discordTester.WaitForLastMessageContains(botUserID, channel.ID, expectedMessage)
		assert.NoError(t, err)
	})

	t.Run("Filters list", func(t *testing.T) {
		command := "filters list"
		expectedMessage := codeBlock(heredoc.Doc(`
			FILTER                  ENABLED DESCRIPTION
			NodeEventsChecker       true    Sends notifications on node level critical events.
			ObjectAnnotationChecker true    Checks if annotations botkube.io/* present in object specs and filters them.`))

		discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
		err := discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
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
			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err := discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With custom cluster name", func(t *testing.T) {
			command := fmt.Sprintf("commands list --cluster-name %s", appCfg.ClusterName)

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("With unknown cluster name", func(t *testing.T) {
			command := "commands list --cluster-name non-existing"

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			t.Log("Ensuring bot didn't write anything new...")
			time.Sleep(appCfg.Discord.MessageWaitTimeout)
			// Same expected message as before
			err = discordTester.WaitForLastMessageContains(testerUserID, channel.ID, command)
			assert.NoError(t, err)
		})
	})

	t.Run("Executor", func(t *testing.T) {
		t.Run("Get Deployment", func(t *testing.T) {
			command := fmt.Sprintf("get deploy -n %s %s", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
			assertionFn := func(msg *discordgo.Message) bool {
				return strings.Contains(msg.Content, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg.Content, "botkube")
			}

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get Configmap", func(t *testing.T) {
			command := fmt.Sprintf("get configmap -n %s", appCfg.Deployment.Namespace)
			assertionFn := func(msg *discordgo.Message) bool {
				return strings.Contains(msg.Content, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
					strings.Contains(msg.Content, "kube-root-ca.crt") &&
					strings.Contains(msg.Content, "botkube-global-config")
			}

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
			assert.NoError(t, err)
		})

		t.Run("Get forbidden resource", func(t *testing.T) {
			command := "get ingress"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'ingress' resources in the 'default' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify unknown command", func(t *testing.T) {
			command := "unknown"
			expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify invalid command", func(t *testing.T) {
			command := "get"
			expectedMessage := codeBlock(fmt.Sprintf("Cluster: %s\nYou must specify the type of resource to get. Use \"kubectl api-resources\" for a complete list of supported resources.\n\nerror: Required resource not specified.\nUse \"kubectl explain <resource>\" for a detailed description of that resource (e.g. kubectl explain pods).\nSee 'kubectl get -h' for help and examples\nexit status 1", appCfg.ClusterName))

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Specify forbidden namespace", func(t *testing.T) {
			command := "get po --namespace team-b"
			expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'po' resources in the 'team-b' Namespace on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

			discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
			err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
			assert.NoError(t, err)
		})

		t.Run("Based on other bindings", func(t *testing.T) {
			t.Run("Wait for Deployment (the 2st binding)", func(t *testing.T) {
				command := fmt.Sprintf("wait deployment -n %s %s --for condition=Available=True", appCfg.Deployment.Namespace, appCfg.Deployment.Name)
				assertionFn := func(msg *discordgo.Message) bool {
					return strings.Contains(msg.Content, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg.Content, "deployment.apps/botkube condition met")
				}

				discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
				err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
				assert.NoError(t, err)
			})

			t.Run("Exec (the 3rd binding which is disabled)", func(t *testing.T) {
				command := "exec"
				expectedMessage := codeBlock("Command not supported. Please run /botkubehelp to see supported commands.")

				discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
				err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Pods (the 4th binding)", func(t *testing.T) {
				command := "get pods -A"
				expectedMessage := codeBlock(fmt.Sprintf("Sorry, the kubectl command is not authorized to work with 'pods' resources for all Namespaces on cluster '%s'. Use 'commands list' to see allowed commands.", appCfg.ClusterName))

				discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
				err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
				assert.NoError(t, err)
			})

			t.Run("Get all Deployments (the 4th binding)", func(t *testing.T) {
				command := "get deploy -A"
				assertionFn := func(msg *discordgo.Message) bool {
					return strings.Contains(msg.Content, heredoc.Doc(fmt.Sprintf("Cluster: %s", appCfg.ClusterName))) &&
						strings.Contains(msg.Content, "local-path-provisioner") &&
						strings.Contains(msg.Content, "coredns") &&
						strings.Contains(msg.Content, "botkube")
				}

				discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
				err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
				assert.NoError(t, err)
			})
		})
	})

	t.Run("Multi-channel notifications", func(t *testing.T) {
		cfgMapCli := k8sCli.CoreV1().ConfigMaps(appCfg.Deployment.Namespace)
		var channelIDs []string
		for _, channel := range channels {
			channelIDs = append(channelIDs, channel.ID)
		}

		t.Log("Creating ConfigMap...")
		var cfgMapAlreadyDeleted bool
		cfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      channel.Name,
				Namespace: appCfg.Deployment.Namespace,
			},
		}
		cfgMap, err = cfgMapCli.Create(context.Background(), cfgMap, metav1.CreateOptions{})
		require.NoError(t, err)

		t.Cleanup(func() { cleanupCreatedCfgMapIfShould(t, cfgMapCli, cfgMap.Name, &cfgMapAlreadyDeleted) })

		t.Log("Expecting bot message in first channel...")
		assertionFn := func(msg *discordgo.Message) bool {
			return doesDiscordMessageContainExactlyOneEmbed(
				msg,
				"v1/configmaps created",
				8311585,
				fmt.Sprintf("ConfigMap *%s/%s* has been created in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Expecting no bot message in second channel...")
		expectedMessage := fmt.Sprintf("...and now my watch begins for cluster '%s'! :crossed_swords:", appCfg.ClusterName)
		time.Sleep(appCfg.Discord.MessageWaitTimeout)
		err = discordTester.WaitForLastMessageEqual(botUserID, secondChannel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Updating ConfigMap...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message in all channels...")
		assertionFn = func(msg *discordgo.Message) bool {
			return doesDiscordMessageContainExactlyOneEmbed(
				msg,
				"v1/configmaps updated",
				16312092,
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = discordTester.WaitForMessagesPostedOnChannels(botUserID, channelIDs, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Stopping notifier...")
		command := "notifier stop"
		expectedMessage = codeBlock(fmt.Sprintf("Sure! I won't send you notifications from cluster '%s' here.", appCfg.ClusterName))

		discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
		err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from second channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are enabled here.", appCfg.ClusterName))
		discordTester.PostMessageToBot(t, botUserID, secondChannel.ID, command)
		err = discordTester.WaitForLastMessageEqual(botUserID, secondChannel.ID, expectedMessage)
		assert.NoError(t, err)

		t.Log("Getting notifier status from first channel...")
		command = "notifier status"
		expectedMessage = codeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", appCfg.ClusterName))
		discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
		err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
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
		err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Expecting bot message on second channel...")
		assertionFn = func(msg *discordgo.Message) bool {
			return doesDiscordMessageContainExactlyOneEmbed(
				msg,
				"v1/configmaps updated",
				16312092,
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = discordTester.WaitForMessagePosted(botUserID, secondChannel.ID, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Starting notifier")
		command = "notifier start"
		expectedMessage = codeBlock(fmt.Sprintf("Brace yourselves, incoming notifications from cluster '%s'.", appCfg.ClusterName))
		discordTester.PostMessageToBot(t, botUserID, channel.ID, command)
		err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Creating and deleting ignored ConfigMap")
		ignoredCfgMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-ignored", channel.Name),
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
		err = discordTester.WaitForLastMessageEqual(botUserID, channel.ID, expectedMessage)
		require.NoError(t, err)

		t.Log("Deleting ConfigMap")
		err = cfgMapCli.Delete(context.Background(), cfgMap.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
		cfgMapAlreadyDeleted = true

		t.Log("Expecting bot message on first channel...")
		assertionFn = func(msg *discordgo.Message) bool {
			return doesDiscordMessageContainExactlyOneEmbed(
				msg,
				"v1/configmaps deleted",
				13632027,
				fmt.Sprintf("ConfigMap *%s/%s* has been deleted in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Ensuring bot didn't write anything new on second channel...")
		time.Sleep(appCfg.Slack.MessageWaitTimeout)
		assertionFn = func(msg *discordgo.Message) bool {
			return doesDiscordMessageContainExactlyOneEmbed(
				msg,
				"v1/configmaps updated",
				16312092,
				fmt.Sprintf("ConfigMap *%s/%s* has been updated in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = discordTester.WaitForMessagePosted(botUserID, secondChannel.ID, 1, assertionFn)
		require.NoError(t, err)
	})

	t.Run("Recommendations", func(t *testing.T) {
		podCli := k8sCli.CoreV1().Pods(appCfg.Deployment.Namespace)

		t.Log("Creating Pod...")
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      channel.Name,
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
		assertionFn := func(msg *discordgo.Message) bool {
			if len(msg.Embeds) != 1 {
				return false
			}

			embed := msg.Embeds[0]
			title := embed.Title
			fieldMessage := embed.Description

			return title == "v1/pods created" &&
				strings.Contains(fieldMessage, "Recommendations:") &&
				strings.Contains(fieldMessage, fmt.Sprintf("- Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name)) &&
				strings.Contains(fieldMessage, fmt.Sprintf("- The 'latest' tag used in '%s' image of Pod '%s/%s' container '%s' should be avoided.", pod.Spec.Containers[0].Image, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name))
		}
		err = discordTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)
	})
}

func codeBlock(in string) string {
	return fmt.Sprintf("```\n%s\n```", in)
}

func doesSlackMessageContainExactlyOneAttachment(msg slack.Message, expectedTitle, expectedColor, expectedFieldMessage string) bool {
	if len(msg.Attachments) != 1 {
		return false
	}

	attachment := msg.Attachments[0]
	title := attachment.Title
	color := attachment.Color

	if len(attachment.Fields) != 1 {
		return false
	}

	fieldMessage := attachment.Fields[0].Value

	return title == expectedTitle &&
		color == expectedColor &&
		fieldMessage == expectedFieldMessage
}

func doesDiscordMessageContainExactlyOneEmbed(msg *discordgo.Message, expectedTitle string, expectedColor int, expectedFieldMessage string) bool {
	if len(msg.Embeds) != 1 {
		return false
	}

	embed := msg.Embeds[0]
	return embed.Title == expectedTitle &&
		embed.Color == expectedColor &&
		embed.Description == expectedFieldMessage
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
