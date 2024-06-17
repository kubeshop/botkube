//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/hasura/go-graphql-client"
	"github.com/mattn/go-shellwords"
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

const (
	// Chromium is not supported by Slack web app for some reason
	// Currently, we get:
	//   This browser won’t be supported starting September 1st, 2024. Update your browser to keep using Slack. Learn more:
	//   https://slack.com/intl/en-gb/help/articles/1500001836081-Slack-support-life-cycle-for-operating-systems-app-versions-and-browsers
	chromeUserAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	authHeaderName            = "Authorization"
	cleanupRetryAttempts      = 5
	awaitInstanceStatusChange = 2 * time.Minute
)

type E2ESlackConfig struct {
	Slack        SlackConfig
	BotkubeCloud BotkubeCloudConfig

	PageTimeout    time.Duration `envconfig:"default=5m"`
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
	t.Log("Loading configuration...")
	var cfg E2ESlackConfig
	err := envconfig.Init(&cfg)
	require.NoError(t, err)

	cfg.Slack.Tester.CloudBasedTestEnabled = false // override property used only in the Cloud Slack E2E tests
	cfg.Slack.Tester.RecentMessagesLimit = 4       // this is used effectively only for the Botkube restarts. There are two of them in a short time window, so it shouldn't be higher than 5.

	authHeaderValue := ""
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

	gqlEndpoint := fmt.Sprintf("%s/%s", cfg.BotkubeCloud.APIBaseURL, cfg.BotkubeCloud.APIGraphQLEndpoint)

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

	connectedDeploy := &gqlModel.Deployment{
		Name: channel.Name(),
	}

	t.Run("Creating Botkube Instance with newly added Slack Workspace", func(t *testing.T) {
		t.Log("Setting up browser...")

		launcher := launcher.New().Headless(false)
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

		t.Log("Log into Botkube Cloud Dashboard")
		page.MustNavigate(appendOrgIDQueryParam(t, cfg.BotkubeCloud.UIBaseURL, cfg.BotkubeCloud.TeamOrganizationID))
		page.MustWaitNavigation()
		page.MustElement(`input[name="username"]`).MustInput(cfg.BotkubeCloud.Email)
		page.MustElement(`input[name="password"]`).MustInput(cfg.BotkubeCloud.Password)
		screenshotIfShould(t, cfg, page)
		page.MustElementR("button", "^Continue$").MustClick()
		screenshotIfShould(t, cfg, page)

		t.Logf("Starting hijacking requests to %q to get the bearer token...", gqlEndpoint)
		router := browser.HijackRequests()
		router.MustAdd(gqlEndpoint, func(ctx *rod.Hijack) {
			if authHeaderValue != "" {
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}

			if ctx.Request != nil && ctx.Request.Method() != http.MethodPost {
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}

			require.NotNil(t, ctx.Request)
			authHeaderValue = ctx.Request.Header(authHeaderName)
			ctx.ContinueRequest(&proto.FetchContinueRequest{})
		})
		go router.Run()
		defer router.MustStop()

		t.Log("Hide Botkube cookie banner")
		page.MustElementR("button", "^Decline$").MustClick()

		t.Log("Create new Botkube Instance")
		page.MustElement("h6#create-instance").MustClick() // case-insensitive
		page.MustElement(`input[name="name"]`).MustSelectAllText().MustInput(connectedDeploy.Name)
		_, connectedDeploy.ID, _ = strings.Cut(page.MustInfo().URL, "add/")
		screenshotIfShould(t, cfg, page)

		installCmd := page.MustElement("div#install-upgrade-cmd > kbd").MustText()

		t.Log("Installing Botkube using Botkube CLI")
		installViaBotkubeCLI(t, cfg.BotkubeCliBinaryPath, installCmd)

		t.Log("Cluster connected...")
		page.MustElement("button#cluster-connected").MustClick()
		page.MustElement(`button[aria-label="Add tab"]`).MustClick()
		page.MustWaitStable()
		page.MustElementR("button", "^Slack$").MustClick()
		page.MustWaitStable()

		page.MustElementR("a", "Add to Slack").MustClick()
		slackPage := browser.MustPages().MustFindByURL("slack.com")

		slackPage.MustElement("input#domain").MustInput(cfg.Slack.WorkspaceName)
		screenshotIfShould(t, cfg, slackPage)
		slackPage.MustElementR("button", "Continue").MustClick()
		screenshotIfShould(t, cfg, slackPage)
		if !launcher.Has(flags.Headless) { // here we get reloaded, so we need to type it again (looks like bug on Slack side)
			slackPage.MustElement("input#domain").MustInput(cfg.Slack.WorkspaceName)
			slackPage.MustElementR("button", "Continue").MustClick()
		}

		slackPage.MustWaitStable()
		slackPage.MustElementR("a", "sign in with a password instead").MustClick()
		screenshotIfShould(t, cfg, slackPage)
		slackPage.MustElement("input#email").MustInput(cfg.Slack.Email)
		slackPage.MustElement("input#password").MustInput(cfg.Slack.Password)
		screenshotIfShould(t, cfg, slackPage)

		t.Log("Hide Slack cookie banner that collides with 'Sign in' button")
		slackPage.MustElement("button#onetrust-accept-btn-handler").MustClick()
		slackPage.MustElementR("button", "/^Sign in$/i").MustClick()
		screenshotIfShould(t, cfg, slackPage)

		slackPage.MustElementR("button.c-button:not(.c-button--disabled)", "Allow").MustClick()

		t.Log("Finalizing Slack workspace connection...")
		if cfg.Slack.WorkspaceAlreadyConnected {
			t.Log("Expecting already connected message...")
			slackPage.MustElementR("div.ant-result-title", "Organization Already Connected!")
			_ = slackPage.Close() // the page should be closed automatically anyway
		} else {
			t.Log("Finalizing connection...")
			screenshotIfShould(t, cfg, slackPage)
			slackPage.MustElement("button#slack-workspace-connect").MustClick()
			screenshotIfShould(t, cfg, slackPage)
			_ = slackPage.Close() // the page should be closed automatically anyway
		}

		if launcher.Has(flags.Headless) {
			// a workaround as the page was often not refreshed with a newly connected Slack Workspace,
			// it only occurs with headless mode
			// TODO(@pkosiec): Do you have a better idea how to fix it?
			t.Log("Re-adding Slack platform")
			page.MustActivate()
			page.MustElement(`button[aria-label="remove"]`).MustClick()
			page.MustElement(`button[aria-label="Add tab"]`).MustClick()
			page.MustElementR("button", "^Slack$").MustClick()
			screenshotIfShould(t, cfg, page)
		}

		t.Logf("Selecting newly connected %q Slack Workspace", cfg.Slack.WorkspaceName)
		page.MustElement(`input[type="search"]`).
			MustInput(cfg.Slack.WorkspaceName).
			MustType(input.Enter)
		screenshotIfShould(t, cfg, page)

		// filter by channel, to make sure that it's visible on the first table page, in order to select it in the next step
		t.Log("Filtering by channel name")
		page.Keyboard.MustType(input.End) // scroll bottom, as the footer collides with selecting filter
		page.MustElement("table th:nth-child(3) span.ant-dropdown-trigger.ant-table-filter-trigger").MustFocus().MustClick()

		t.Log("Selecting channel checkbox")
		page.MustElement("input#name-channel").MustInput(channel.Name()).MustType(input.Enter)
		page.MustElement(fmt.Sprintf(`input[type="checkbox"][name="%s"]`, channel.Name())).MustClick()

		t.Log("Navigating to plugin selection")
		page.MustElementR("button", "/^Next$/i").MustClick().MustWaitStable()

		t.Log("Using pre-selected plugins. Navigating to wizard summary")
		page.MustElementR("button", "/^Next$/i").MustClick().MustWaitStable()

		t.Log("Submitting changes")
		page.MustElementR("button", "/^Deploy changes$/i").MustClick().MustWaitStable()

		t.Log("Waiting for status 'Connected'")
		page.Timeout(awaitInstanceStatusChange).MustElementR("div#deployment-status", "Connected")

		//
		t.Log("Updating 'kubectl' namespace property")
		page.MustElementR(`div[role="tab"]`, "Plugins").MustClick()
		page.MustElement(`button[id^="botkube/kubectl_"]`).MustClick()
		page.MustElement(`div[data-node-key="ui-form"]`).MustClick()
		page.MustElementR("input#root_defaultNamespace", "default").MustSelectAllText().MustInput("kube-system")
		page.MustElementR("button", "/^Update$/i").MustClick()

		t.Log("Submitting changes")
		page.MustElementR("button", "/^Deploy changes$/i").MustClick().MustWaitStable() // use the case-insensitive flag "i"

		t.Log("Waiting for status 'Updating'")
		page.Timeout(awaitInstanceStatusChange).MustElementR("div#deployment-status", "Updating")

		t.Log("Waiting for status 'Connected'")
		page.Timeout(awaitInstanceStatusChange).MustElementR("div#deployment-status", "Connected")

		t.Log("Verifying that the 'namespace' value was updated and persisted properly")
		page.MustElementR(`div[role="tab"]`, "Plugins").MustClick()
		page.MustElement(`button[id^="botkube/kubectl_"]`).MustClick()
		page.MustElement(`div[data-node-key="ui-form"]`).MustClick()
		page.MustElementR("input#root_defaultNamespace", "kube-system")
	})

	t.Run("Run E2E tests with deployment", func(t *testing.T) {
		require.NotEmpty(t, authHeaderValue, "Previous subtest needs to pass to get authorization header value")

		fmt.Println(authHeaderValue)
		t.Logf("Using Organization ID %q and Authorization header starting with %q", cfg.BotkubeCloud.TeamOrganizationID,
			stringsutil.ShortenString(authHeaderValue, 15))

		gqlCli := cloud_graphql.NewClientForAuthAndOrg(gqlEndpoint, cfg.BotkubeCloud.TeamOrganizationID, authHeaderValue)

		t.Logf("Getting connected Slack workspace...")
		slackWorkspaces := gqlCli.MustListSlackWorkspacesForOrg(t, cfg.BotkubeCloud.TeamOrganizationID)
		require.Len(t, slackWorkspaces, 1)
		slackWorkspace := slackWorkspaces[0]
		require.NotNil(t, slackWorkspace)
		t.Cleanup(func() {
			if !cfg.Slack.DisconnectWorkspaceAfterTests {
				return
			}
			t.Log("Disconnecting Slack workspace...")
			err = retryOperation(func() error {
				return gqlCli.DeleteSlackWorkspace(t, cfg.BotkubeCloud.TeamOrganizationID, slackWorkspace.ID)
			})
			if err != nil {
				t.Logf("Failed to disconnect Slack workspace: %s", err.Error())
			}
		})

		t.Log("Creating a second deployment to test not connected flow...")
		notConnectedDeploy := gqlCli.MustCreateBasicDeploymentWithCloudSlack(t, fmt.Sprintf("%s-2", channel.Name()), slackWorkspace.TeamID, channel.Name())
		t.Cleanup(func() {
			t.Log("Deleting second deployment...")
			err = retryOperation(func() error {
				return gqlCli.DeleteDeployment(t, graphql.ID(notConnectedDeploy.ID))
			})
			if err != nil {
				t.Logf("Failed to delete second deployment: %s", err.Error())
			}
		})

		t.Cleanup(func() {
			t.Log("Deleting first deployment...")
			err = retryOperation(func() error {
				return gqlCli.DeleteDeployment(t, graphql.ID(connectedDeploy.ID))
			})
			if err != nil {
				t.Logf("Failed to delete first deployment: %s", err.Error())
			}
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

			t.Log("Waiting for config reload message...")
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", connectedDeploy.Name)
			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log("Waiting for watch begin message...")
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

			t.Log("Waiting for config reload message...")
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", connectedDeploy.Name)
			tester.SetTimeout(90 * time.Second)
			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log("Waiting for watch begin message...")
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

func installViaBotkubeCLI(t *testing.T, botkubeBinary, installCmd string) {
	args, err := shellwords.Parse(installCmd)
	args = append(args, "--auto-approve")
	require.NoError(t, err)

	cmd := exec.Command(botkubeBinary, args[1:]...)
	installOutput, err := cmd.CombinedOutput()
	t.Log(string(installOutput))
	require.NoError(t, err)
}

func newBrowserPage(t *testing.T, browser *rod.Browser, cfg E2ESlackConfig) *rod.Page {
	t.Helper()

	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	require.NoError(t, err)
	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: chromeUserAgent,
	})
	page = page.Timeout(cfg.PageTimeout)
	page.MustSetViewport(1200, 1080, 1, false)
	return page
}

func removeSourcesAndAddActions(t *testing.T, gql *graphql.Client, existingDeployment *gqlModel.Deployment) *gqlModel.Deployment {
	var updateInput struct {
		UpdateDeployment gqlModel.Deployment `graphql:"updateDeployment(id: $id, input: $input)"`
	}
	var updatePluginGroup []*gqlModel.PluginConfigurationGroupUpdateInput
	for _, createdPlugin := range existingDeployment.Plugins {
		var pluginConfigs []*gqlModel.PluginConfigurationUpdateInput
		pluginConfig := gqlModel.PluginConfigurationUpdateInput{
			Name:          createdPlugin.ConfigurationName,
			Configuration: createdPlugin.Configuration,
		}
		pluginConfigs = append(pluginConfigs, &pluginConfig)
		plugin := gqlModel.PluginConfigurationGroupUpdateInput{
			ID:             &createdPlugin.ID,
			Name:           createdPlugin.Name,
			Type:           createdPlugin.Type,
			DisplayName:    createdPlugin.DisplayName,
			Configurations: pluginConfigs,
		}
		updatePluginGroup = append(updatePluginGroup, &plugin)
	}
	var updatePlugins []*gqlModel.PluginsUpdateInput
	updatePlugin := gqlModel.PluginsUpdateInput{
		Groups: updatePluginGroup,
	}
	updatePlugins = append(updatePlugins, &updatePlugin)

	platforms := gqlModel.PlatformsUpdateInput{}

	for _, slack := range existingDeployment.Platforms.CloudSlacks {
		var channelUpdateInputs []*gqlModel.ChannelBindingsByNameAndIDUpdateInput
		for _, channel := range slack.Channels {
			channelUpdateInputs = append(channelUpdateInputs, &gqlModel.ChannelBindingsByNameAndIDUpdateInput{
				ChannelID: "", // this is used for UI only so we don't need to provide it
				Name:      channel.Name,
				Bindings: &gqlModel.BotBindingsUpdateInput{
					Sources:   nil,
					Executors: []*string{&channel.Bindings.Executors[0]},
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
			Plugins:         updatePlugins,
			Platforms:       &platforms,
			Actions:         CreateActionUpdateInput(existingDeployment),
		},
	}
	err := gql.Mutate(context.Background(), &updateInput, updateVariables)
	require.NoError(t, err)

	return &updateInput.UpdateDeployment
}

func screenshotIfShould(t *testing.T, cfg E2ESlackConfig, page *rod.Page) {
	t.Helper()
	if cfg.ScreenshotsDir == "" {
		return
	}

	pathParts := strings.Split(cfg.ScreenshotsDir, "/")
	pathParts = append(pathParts)

	filePath := filepath.Join(cfg.ScreenshotsDir, fmt.Sprintf("%d.png", time.Now().UnixNano()))

	logMsg := fmt.Sprintf("Saving screenshot to %q", filePath)
	if cfg.DebugMode {
		info, err := page.Info()
		assert.NoError(t, err)

		if info != nil {
			logMsg += fmt.Sprintf(" for URL %q", info.URL)
		}
	}
	t.Log(logMsg)
	data, err := page.Screenshot(true, nil)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	err = os.WriteFile(filePath, data, 0o644)
	assert.NoError(t, err)
}

func appendOrgIDQueryParam(t *testing.T, inURL, orgID string) string {
	parsedURL, err := url.Parse(inURL)
	require.NoError(t, err)
	queryValues := parsedURL.Query()
	queryValues.Set("organizationId", orgID)
	parsedURL.RawQuery = queryValues.Encode()

	return parsedURL.String()
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

func closePage(t *testing.T, name string, page *rod.Page) {
	t.Helper()
	err := page.Close()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}

		t.Logf("Failed to close page %q: %v", name, err)
	}
}

func retryOperation(fn func() error) error {
	return retry.Do(fn,
		retry.Attempts(cleanupRetryAttempts),
		retry.Delay(500*time.Millisecond),
		retry.LastErrorOnly(false),
	)
}
