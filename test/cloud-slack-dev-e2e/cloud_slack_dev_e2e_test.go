//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"botkube.io/botube/test/botkubex"
	"botkube.io/botube/test/cloud_graphql"
	"botkube.io/botube/test/commplatform"
	"botkube.io/botube/test/diff"
	"github.com/avast/retry-go/v4"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/hasura/go-graphql-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	stringsutil "k8s.io/utils/strings"

	gqlModel "github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/graphql"
	"github.com/kubeshop/botkube/pkg/formatx"
)

const cleanupRetryAttempts = 5

type E2ESlackConfig struct {
	Slack        SlackConfig
	BotkubeCloud BotkubeCloudConfig

	PageTimeout    time.Duration `envconfig:"default=10m"`
	ScreenshotsDir string        `envconfig:"optional"`
	DebugMode      bool          `envconfig:"default=false"`

	ClusterNamespace     string `envconfig:"default=default"`
	Kubeconfig           string `envconfig:"optional"`
	BotkubeCliBinaryPath string
}

type SlackConfig struct {
	WorkspaceName                 string
	Email                         string
	Password                      string
	WorkspaceAlreadyConnected     bool `envconfig:"default=false"`
	DisconnectWorkspaceAfterTests bool `envconfig:"default=true"`

	Tester commplatform.SlackConfig
}

type BotkubeCloudConfig struct {
	APIBaseURL         string `envconfig:"default=https://api-dev.botkube.io"`
	UIBaseURL          string `envconfig:"default=https://app-dev.botkube.io"`
	APIGraphQLEndpoint string `envconfig:"default=graphql"`
	Email              string
	Password           string

	TeamOrganizationID string
}

func TestCloudSlackE2E(t *testing.T) {
	t.Log("1. Loading configuration...")
	var cfg E2ESlackConfig
	err := envconfig.Init(&cfg)
	require.NoError(t, err)

	cfg.Slack.Tester.CloudBasedTestEnabled = false        // override property used only in the Cloud Slack E2E tests
	cfg.Slack.Tester.RecentMessagesLimit = 3              // this is used effectively only for the Botkube restarts. There are two of them in a short time window, so it shouldn't be higher than 5.
	cfg.Slack.Tester.MessageWaitTimeout = 3 * time.Minute // downloading plugins on restarted Agents, sometimes takes a while on GitHub runners.

	var botkubeDeploymentUninstalled atomic.Bool
	botkubeDeploymentUninstalled.Store(true) // not yet installed
	t.Cleanup(func() {
		if t.Failed() {
			t.Log("Tests failed, keeping the Botkube instance installed for debugging purposes.")
			return
		}
		if botkubeDeploymentUninstalled.Load() {
			return
		}
		t.Log("Uninstalling Botkube...")
		botkubex.Uninstall(t, cfg.BotkubeCliBinaryPath)

		botkubeDeploymentUninstalled.Store(true)
	})

	if cfg.ScreenshotsDir != "" {
		t.Logf("Screenshots enabled. They will be saved to %s", cfg.ScreenshotsDir)
		err = os.MkdirAll(cfg.ScreenshotsDir, os.ModePerm)
		require.NoError(t, err)
	} else {
		t.Log("Screenshots disabled.")
	}

	t.Log("Initializing Slack...")
	tester, err := commplatform.NewSlackTester(cfg.Slack.Tester, nil)
	require.NoError(t, err)

	t.Log("Initializing users...")
	tester.InitUsers(t)

	t.Log("Creating channel...")
	channel, createChannelCallback := tester.CreateChannel(t, "e2e-test")
	t.Cleanup(func() { createChannelCallback(t) })

	t.Log("Inviting Bot to the channel...")
	tester.InviteBotToChannel(t, channel.ID())

	botkubeCloudPage := NewBotkubeCloudPage(t, cfg)
	slackPage := NewSlackPage(t, cfg)

	t.Log("2. Creating Botkube Instance with newly added Slack Workspace")

	t.Log("Setting up browser...")
	launcher := launcher.New().Headless(true)
	isHeadless := launcher.Has(flags.Headless)
	t.Cleanup(launcher.Cleanup)

	browser := rod.New().Trace(cfg.DebugMode).ControlURL(launcher.MustLaunch()).MustConnect()
	t.Cleanup(func() {
		err := browser.Close()
		if err != nil {
			t.Logf("Failed to close browser: %v", err)
		}
	})

	page := newBrowserPage(t, browser, cfg)
	t.Cleanup(func() {
		closePage(t, "page", page)
	})

	stopRouter := botkubeCloudPage.InterceptBearerToken(t, browser)
	defer stopRouter()

	botkubeCloudPage.NavigateAndLogin(t, page)
	botkubeCloudPage.HideCookieBanner(t)

	botkubeCloudPage.CreateNewInstance(t, channel.Name())
	t.Cleanup(func() {
		// Delete Botkube instance.
		// Cleanup is skipped if the instance was already deleted.
		// This cleanup is needed if there's a fail between instance creation and Slack workspace connection.
		gqlCli := createGQLCli(t, cfg, botkubeCloudPage)
		botkubeCloudPage.Cleanup(t, gqlCli)
	})
	t.FailNow()
	botkubeCloudPage.InstallAgentInCluster(t, cfg.BotkubeCliBinaryPath)
	botkubeCloudPage.OpenSlackAppIntegrationPage(t)

	slackPage.ConnectWorkspace(t, browser)
	t.Cleanup(func() {
		// Disconnect Slack workspace.
		gqlCli := createGQLCli(t, cfg, botkubeCloudPage)
		slackPage.Cleanup(t, gqlCli)
	})
	t.Cleanup(func() {
		// Delete Botkube instance.
		// The code is repeated on purpose: we want to make sure the instance is cleaned up before the Slack workspace.
		// t.Cleanup functions are called in last added, first called order.
		gqlCli := createGQLCli(t, cfg, botkubeCloudPage)
		botkubeCloudPage.Cleanup(t, gqlCli)
	})

	botkubeCloudPage.ReAddSlackPlatformIfShould(t, isHeadless)
	botkubeCloudPage.SetupSlackWorkspace(t, channel.Name())
	botkubeCloudPage.FinishWizard(t)
	botkubeCloudPage.VerifyDeploymentStatus(t, "Connected")

	botkubeCloudPage.UpdateKubectlNamespace(t)
	botkubeCloudPage.VerifyDeploymentStatus(t, "Updating")
	botkubeCloudPage.VerifyDeploymentStatus(t, "Connected")
	botkubeCloudPage.VerifyUpdatedKubectlNamespace(t)

	t.Run("Run E2E tests with deployment", func(t *testing.T) {
		gqlCli := createGQLCli(t, cfg, botkubeCloudPage)

		connectedDeploy := botkubeCloudPage.ConnectedDeploy
		require.NotNil(t, connectedDeploy, "Previous subtest needs to pass to get connected deployment information")
		// cleanup is done in the upper test function

		slackWorkspace := findConnectedSlackWorkspace(t, cfg, gqlCli)
		require.NotNil(t, slackWorkspace)
		// cleanup is done in the upper test function

		t.Log("Creating a second deployment to test not connected flow...")
		notConnectedDeploy := gqlCli.MustCreateBasicDeploymentWithCloudSlack(t, fmt.Sprintf("%s-2", channel.Name()), slackWorkspace.TeamID, channel.Name())
		t.Cleanup(func() {
			deleteDeployment(t, gqlCli, notConnectedDeploy.ID, "second (not connected)")
		})

		t.Log("Waiting for help message...")
		assertionFn := func(msg string) (bool, int, string) {
			return strings.Contains(msg, fmt.Sprintf("Botkube instance %q is now active.", connectedDeploy.Name)), 0, ""
		}
		err = tester.WaitForMessagePosted(tester.BotUserID(), channel.ID(), 10, assertionFn) // we perform a few restarts before
		require.NoError(t, err)

		cmdHeader := func(command string) string {
			return fmt.Sprintf("`%s` on `%s`", command, connectedDeploy.Name)
		}

		t.Run("Check basic commands", func(t *testing.T) {
			t.Log("Testing ping with --cluster-name")
			command := fmt.Sprintf("ping --cluster-name %s", connectedDeploy.Name)
			expectedMessage := fmt.Sprintf("`%s` on `%s`\n```\npong", command, connectedDeploy.Name)
			tester.PostMessageToBot(t, channel.Identifier(), command)
			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)

			t.Log("Testing ping for not connected deployment #2")
			command = "ping"
			expectedMessage = fmt.Sprintf("The cluster %s (id: %s) is not connected.", notConnectedDeploy.Name, notConnectedDeploy.ID)
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("%s --cluster-name %s", command, notConnectedDeploy.Name))

			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)

			t.Log("Testing ping for not existing deployment")
			command = "ping"
			deployName := "non-existing-deployment"
			expectedMessage = fmt.Sprintf("*Instance not found* The cluster %q does not exist.", deployName)
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("%s --cluster-name %s", command, deployName))
			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)

			t.Log("Setting cluster as default")
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("cloud set default-instance %s", connectedDeploy.ID))
			t.Log("Waiting for confirmation message...")
			expectedClusterDefaultMsg := fmt.Sprintf(":white_check_mark: Instance %s was successfully selected as the default cluster for this channel.", connectedDeploy.Name)
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedClusterDefaultMsg)
			require.NoError(t, err)

			t.Log("Testing getting all deployments")
			command = "kubectl get deployments -A"
			assertionFn := func(msg string) (bool, int, string) {
				return strings.Contains(msg, cmdHeader(command)) &&
					strings.Contains(msg, "coredns") &&
					strings.Contains(msg, "botkube"), 0, ""
			}
			tester.PostMessageToBot(t, channel.Identifier(), command)
			err = tester.WaitForMessagePosted(tester.BotUserID(), channel.ID(), 1, assertionFn)
			require.NoError(t, err)
		})

		t.Run("Get notifications", func(t *testing.T) {
			t.Log("Creating K8s client...")
			k8sCli := createK8sCli(t, cfg.Kubeconfig)

			t.Log("Creating Pod which should trigger recommendations")
			podCli := k8sCli.CoreV1().Pods(cfg.ClusterNamespace)
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      channel.Name(),
					Namespace: cfg.ClusterNamespace,
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

			assertionFn := func(msg string) (bool, int, string) {
				expStrings := []string{
					"*:large_green_circle: v1/pods created*",
					fmt.Sprintf("*Name:* %s", pod.Name),
					fmt.Sprintf("*Namespace:* %s", pod.Namespace),
					fmt.Sprintf("*Cluster:* %s", connectedDeploy.Name),
					"*Recommendations*",
					fmt.Sprintf("Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name),
					fmt.Sprintf("The 'latest' tag used in 'nginx:latest' image of Pod '%s/%s' container 'nginx' should be avoided.", pod.Namespace, pod.Name),
				}

				result := true
				for _, str := range expStrings {
					if !strings.Contains(msg, str) {
						result = false
						t.Logf("Expected string not found in message: %s", str)
					}
				}
				return result, 0, ""
			}
			err = tester.OnChannel().WaitForMessagePosted(tester.BotUserID(), channel.ID(), 1, assertionFn)
			require.NoError(t, err)
		})

		t.Run("Botkube Deployment -> Cloud sync", func(t *testing.T) {
			t.Log("Disabling notification...")
			tester.PostMessageToBot(t, channel.Identifier(), "disable notifications")

			t.Log(time.Now().Format(time.TimeOnly), "Waiting for config reload message...")
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", connectedDeploy.Name)
			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log(time.Now().Format(time.TimeOnly), "Waiting for watch begin message...")
			expectedWatchBeginMsg := fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", connectedDeploy.Name)
			recentMessages := 2 // take into the account the optional "upgrade checker message"
			err = tester.OnChannel().WaitForMessagePosted(tester.BotUserID(), channel.ID(), recentMessages, func(msg string) (bool, int, string) {
				if !strings.EqualFold(expectedWatchBeginMsg, msg) {
					count := diff.CountMatchBlock(expectedWatchBeginMsg, msg)
					msgDiff := diff.Diff(expectedWatchBeginMsg, msg)
					return false, count, msgDiff
				}
				return true, 0, ""
			})
			require.NoError(t, err)

			t.Log("Verifying disabled notification on Cloud...")
			deploy := gqlCli.MustGetDeployment(t, graphql.ID(connectedDeploy.ID))
			require.True(t, *deploy.Platforms.CloudSlacks[0].Channels[0].NotificationsDisabled)

			t.Log("Verifying disabled notifications on chat...")
			command := "status notifications"
			expectedBody := formatx.CodeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", connectedDeploy.Name))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
			tester.PostMessageToBot(t, channel.Identifier(), "status notifications")
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)
		})

		t.Run("Cloud -> Botkube Deployment sync", func(t *testing.T) {
			t.Log("Removing source binding from Slack platform & add actions")
			d := gqlCli.MustGetDeployment(t, graphql.ID(connectedDeploy.ID)) // Get final resource version
			connectedDeploy = removeSourcesAndAddActions(t, gqlCli.Client, &d)

			t.Log(time.Now().Format(time.TimeOnly), "Waiting for config reload message...")
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", connectedDeploy.Name)
			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log(time.Now().Format(time.TimeOnly), "Waiting for watch begin message...")
			expectedWatchBeginMsg := fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", connectedDeploy.Name)
			recentMessages := 2 // take into the account the  optional "upgrade checker message"
			err = tester.OnChannel().WaitForMessagePosted(tester.BotUserID(), channel.ID(), recentMessages, func(msg string) (bool, int, string) {
				if !strings.EqualFold(expectedWatchBeginMsg, msg) {
					count := diff.CountMatchBlock(expectedWatchBeginMsg, msg)
					msgDiff := diff.Diff(expectedWatchBeginMsg, msg)
					return false, count, msgDiff
				}
				return true, 0, ""
			})
			require.NoError(t, err)
			tester.PostMessageToBot(t, channel.Identifier(), "list sources")

			t.Log("Waiting for empty source list...")
			expectedSourceListMsg := fmt.Sprintf("%s\n```\nSOURCE ENABLED RESTARTS STATUS LAST_RESTART\n```", cmdHeader("list sources"))
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedSourceListMsg)
			require.NoError(t, err)
			tester.PostMessageToBot(t, channel.Identifier(), "list actions")
			t.Log("Waiting for actions list...")
			expectedActionsListMsg := fmt.Sprintf("%s\n```\nACTION       ENABLED  DISPLAY NAME\naction_xxx22 true     Action Name\n```", cmdHeader("list actions"))
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedActionsListMsg)
			require.NoError(t, err)
		})

		t.Run("Executed commands and events are audited", func(t *testing.T) {
			var auditPage struct {
				Audits AuditEventPage `graphql:"auditEvents(filter: $filter, offset: $offset, limit: $limit)"`
			}
			variables := map[string]interface{}{
				"offset": 0,
				"limit":  10,
				"filter": gqlModel.AuditEventFilter{
					DeploymentID: &connectedDeploy.ID,
				},
			}

			err := gqlCli.Query(context.Background(), &auditPage, variables)
			require.NoError(t, err)
			require.NotEmpty(t, auditPage.Audits.Data)

			t.Log("Asserting command executed events...")
			botPlatform := gqlModel.BotPlatformSLACk
			want := ExpectedCommandExecutedEvents([]string{
				"kubectl get deployments -A",
				fmt.Sprintf("ping --cluster-name %s", connectedDeploy.Name),
				"disable notifications",
				"status notifications",
				"list sources",
				"list actions",
			}, &botPlatform, channel.Name())

			got := CommandExecutedEventsFromAuditResponse(auditPage.Audits)
			require.ElementsMatch(t, want, got)

			t.Log("Asserting source emitted events...")
			deploy := gqlCli.MustGetDeployment(t, graphql.ID(connectedDeploy.ID))
			source, _ := DeploymentSourceAndExecutor(&deploy)
			wantSrcEvents := []gqlModel.SourceEventEmittedEvent{
				{
					Source: &gqlModel.SourceEventDetails{
						Name:        source,
						DisplayName: "kubernetes",
					},
					PluginName: "botkube/kubernetes",
				},
			}
			gotSrcEvents := SourceEmittedEventsFromAuditResponse(auditPage.Audits)
			require.ElementsMatch(t, wantSrcEvents, gotSrcEvents)
		})
	})
}

func removeSourcesAndAddActions(t *testing.T, gql *graphql.Client, existingDeployment *gqlModel.Deployment) *gqlModel.Deployment {
	var updateInput struct {
		UpdateDeployment gqlModel.Deployment `graphql:"updateDeployment(id: $id, input: $input)"`
	}

	var updatePluginGroup []*gqlModel.PluginConfigurationGroupUpdateInput
	for _, createdPlugin := range existingDeployment.Plugins {
		updatePluginGroup = append(updatePluginGroup, &gqlModel.PluginConfigurationGroupUpdateInput{
			ID:          &createdPlugin.ID,
			Name:        createdPlugin.Name,
			Type:        createdPlugin.Type,
			DisplayName: createdPlugin.DisplayName,
			Configurations: []*gqlModel.PluginConfigurationUpdateInput{
				{
					Name:          createdPlugin.ConfigurationName,
					Configuration: createdPlugin.Configuration,
				},
			},
		})
	}

	platforms := gqlModel.PlatformsUpdateInput{}
	for _, slack := range existingDeployment.Platforms.CloudSlacks {
		var channelUpdateInputs []*gqlModel.ChannelBindingsByNameAndIDUpdateInput
		for _, channel := range slack.Channels {
			channelUpdateInputs = append(channelUpdateInputs, &gqlModel.ChannelBindingsByNameAndIDUpdateInput{
				ChannelID: "", // this is used for UI only so we don't need to provide it
				Name:      channel.Name,
				Bindings: &gqlModel.BotBindingsUpdateInput{
					Sources:   []*string{},
					Executors: []*string{},
				},
			})
		}
		platforms.CloudSlacks = append(platforms.CloudSlacks, &gqlModel.CloudSlackUpdateInput{
			ID:       &slack.ID,
			Name:     slack.Name,
			TeamID:   slack.TeamID,
			Channels: channelUpdateInputs,
		})
	}

	updateVariables := map[string]interface{}{
		"id": graphql.ID(existingDeployment.ID),
		"input": gqlModel.DeploymentUpdateInput{
			Name:            existingDeployment.Name,
			ResourceVersion: existingDeployment.ResourceVersion,
			Plugins: []*gqlModel.PluginsUpdateInput{
				{Groups: updatePluginGroup},
			},
			Platforms: &platforms,
			Actions:   CreateActionUpdateInput(existingDeployment),
		},
	}
	err := gql.Mutate(context.Background(), &updateInput, updateVariables)
	require.NoError(t, err)

	return &updateInput.UpdateDeployment
}

func cleanupCreatedPod(t *testing.T, podCli corev1.PodInterface, name string) {
	t.Log("Cleaning up created Pod...")
	err := podCli.Delete(context.Background(), name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func createK8sCli(t *testing.T, kubeconfigPath string) *kubernetes.Clientset {
	if kubeconfigPath == "" {
		home := homedir.HomeDir()
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	require.NoError(t, err)
	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(t, err)
	return k8sCli
}

func createGQLCli(t *testing.T, cfg E2ESlackConfig, botkubeCloudPage *BotkubeCloudPage) *cloud_graphql.Client {
	require.NotEmpty(t, botkubeCloudPage.AuthHeaderValue, "Authorization header value should be set")

	t.Logf("Using Organization ID %q and Authorization header starting with %q", cfg.BotkubeCloud.TeamOrganizationID,
		stringsutil.ShortenString(botkubeCloudPage.AuthHeaderValue, 15))
	return cloud_graphql.NewClientForAuthAndOrg(botkubeCloudPage.GQLEndpoint, cfg.BotkubeCloud.TeamOrganizationID, botkubeCloudPage.AuthHeaderValue)
}

func findConnectedSlackWorkspace(t *testing.T, cfg E2ESlackConfig, gqlCli *cloud_graphql.Client) *gqlModel.SlackWorkspace {
	t.Logf("Finding connected Slack workspace...")
	slackWorkspaces := gqlCli.MustListSlackWorkspacesForOrg(t, cfg.BotkubeCloud.TeamOrganizationID)
	if len(slackWorkspaces) == 0 {
		return nil
	}

	if len(slackWorkspaces) > 1 {
		t.Logf("Found multiple connected Slack workspaces: %v", slackWorkspaces)
		return nil
	}

	slackWorkspace := slackWorkspaces[0]
	return slackWorkspace
}

func disconnectConnectedSlackWorkspace(t *testing.T, cfg E2ESlackConfig, gqlCli *cloud_graphql.Client, slackWorkspace *gqlModel.SlackWorkspace) {
	if slackWorkspace == nil {
		t.Log("Skipping disconnecting Slack workspace as it is nil")
		return
	}

	if !cfg.Slack.DisconnectWorkspaceAfterTests {
		t.Log("Skipping disconnecting Slack workspace...")
		return
	}

	t.Log("Disconnecting Slack workspace...")
	err := retryOperation(func() error {
		return gqlCli.DeleteSlackWorkspace(t, cfg.BotkubeCloud.TeamOrganizationID, slackWorkspace.ID)
	})
	if err != nil {
		t.Logf("Failed to disconnect Slack workspace: %s", err.Error())
	}
}

func deleteDeployment(t *testing.T, gqlCli *cloud_graphql.Client, deployID string, label string) {
	t.Logf("Deleting %s deployment...", label)
	err := retryOperation(func() error {
		return gqlCli.DeleteDeployment(t, graphql.ID(deployID))
	})
	if err != nil {
		t.Logf("Failed to delete first deployment: %s", err.Error())
	}
}

func retryOperation(fn func() error) error {
	return retry.Do(fn,
		retry.Attempts(cleanupRetryAttempts),
		retry.Delay(500*time.Millisecond),
		retry.LastErrorOnly(false),
	)
}
