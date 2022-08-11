//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/pkg/filterengine/filters"
)

type Config struct {
	KubeconfigPath string `envconfig:"optional,KUBECONFIG"`
	Deployment     struct {
		Name          string        `envconfig:"default=botkube"`
		Namespace     string        `envconfig:"default=botkube"`
		ContainerName string        `envconfig:"default=botkube"`
		WaitTimeout   time.Duration `envconfig:"default=3m"`
		Envs          struct {
			SlackEnabledName            string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_ENABLED"`
			DefaultSlackChannelIDName   string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_CHANNELS_DEFAULT_NAME"`
			SecondarySlackChannelIDName string `envconfig:"default=BOTKUBE_COMMUNICATIONS_DEFAULT-GROUP_SLACK_CHANNELS_SECONDARY_NAME"`
		}
	}
	ClusterName string `envconfig:"default=sample"`
	Slack       SlackConfig
}

type SlackConfig struct {
	BotName                  string `envconfig:"default=botkube"`
	TesterName               string `envconfig:"default=tester"`
	AdditionalContextMessage string `envconfig:"optional"`
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
	revertDeployFn := setTestEnvsForDeploy(t, appCfg, deployNsCli, channels)
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
			ImageTagChecker         true    Checks and adds recommendation if 'latest' image tag is used for container image.
			IngressValidator        true    Checks if services and tls secrets used in ingress specs are available.
			NamespaceChecker        true    Checks if event belongs to blocklisted namespaces and filter them.
			NodeEventsChecker       true    Sends notifications on node level critical events.
			ObjectAnnotationChecker true    Checks if annotations <http://botkube.io/*|botkube.io/*> present in object specs and filters them.
			PodLabelChecker         true    Checks and adds recommendations if labels are missing in the pod specs.`))

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
						strings.Contains(msg.Text, "traefik") &&
						strings.Contains(msg.Text, "metrics-server") &&
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

		t.Log("Expecting bot message...")
		assertionFn := func(msg slack.Message) bool {
			return doesMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps created",
				"2eb886",
				fmt.Sprintf("ConfigMap *%s/%s* has been created in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagesPostedOnChannels(botUserID, channelIDs, 1, assertionFn)
		require.NoError(t, err)

		t.Log("Updating ConfigMap...")
		cfgMap.Data = map[string]string{
			"operation": "update",
		}
		cfgMap, err = cfgMapCli.Update(context.Background(), cfgMap, metav1.UpdateOptions{})
		require.NoError(t, err)

		t.Log("Expecting bot message...")
		assertionFn = func(msg slack.Message) bool {
			return doesMessageContainExactlyOneAttachment(
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
		expectedMessage := codeBlock(fmt.Sprintf("Sure! I won't send you notifications from cluster '%s' here.", appCfg.ClusterName))

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
			return doesMessageContainExactlyOneAttachment(
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

		t.Log("Expecting bot message...")
		assertionFn = func(msg slack.Message) bool {
			return doesMessageContainExactlyOneAttachment(
				msg,
				"v1/configmaps deleted",
				"a30200",
				fmt.Sprintf("ConfigMap *%s/%s* has been deleted in *%s* cluster", cfgMap.Namespace, cfgMap.Name, appCfg.ClusterName),
			)
		}
		err = slackTester.WaitForMessagesPostedOnChannels(botUserID, channelIDs, 1, assertionFn)
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
				strings.Contains(fieldMessage, "- :latest tag used in image 'nginx:latest' of Container 'nginx' should be avoided.") &&
				strings.Contains(fieldMessage, fmt.Sprintf("- pod '%s' creation without labels should be avoided.", pod.Name))
		}
		err = slackTester.WaitForMessagePosted(botUserID, channel.ID, 1, assertionFn)
		require.NoError(t, err)
	})
}

func codeBlock(in string) string {
	return fmt.Sprintf("```\n%s\n```", in)
}

func doesMessageContainExactlyOneAttachment(msg slack.Message, expectedTitle, expectedColor, expectedFieldMessage string) bool {
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
