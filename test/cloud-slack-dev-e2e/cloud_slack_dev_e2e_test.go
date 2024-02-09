//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"botkube.io/botube/test/cloud_graphql"
	"botkube.io/botube/test/commplatform"
	"botkube.io/botube/test/diff"
	"botkube.io/botube/test/helmx"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
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

const (
	// Chromium is not supported by Slack web app for some reason
	chromeUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	authHeaderName  = "Authorization"
)

type E2ESlackConfig struct {
	Slack        SlackConfig
	BotkubeCloud BotkubeCloudConfig

	PageTimeout    time.Duration `envconfig:"default=1m"`
	ScreenshotsDir string        `envconfig:"optional"`
	DebugMode      bool          `envconfig:"default=false"`

	ClusterNamespace string        `envconfig:"default=default"`
	Kubeconfig       string        `envconfig:"optional"`
	DefaultWaitTime  time.Duration `envconfig:"default=10s"`
}

type SlackConfig struct {
	WorkspaceName                 string
	Email                         string
	Password                      string
	BotDisplayName                string `envconfig:"default=BotkubeDev"`
	ConversationWithBotURL        string `envconfig:"default=https://app.slack.com/client/"`
	WorkspaceAlreadyConnected     bool   `envconfig:"default=false"`
	DisconnectWorkspaceAfterTests bool   `envconfig:"default=true"`

	Tester commplatform.SlackConfig
}

type BotkubeCloudConfig struct {
	APIBaseURL                             string `envconfig:"default=https://api-dev.botkube.io"`
	APIGraphQLEndpoint                     string `envconfig:"default=graphql"`
	APISlackAppInstallationBaseURLOverride string `envconfig:"optional"`
	APISlackAppInstallationEndpoint        string `envconfig:"default=routers/slack/v1/install"`
	Email                                  string
	Password                               string

	TeamOrganizationID string
	FreeOrganizationID string
	PluginRepoURL      string `envconfig:"default=https://storage.googleapis.com/botkube-plugins-latest/plugins-dev-index.yaml"`
}

func TestCloudSlackE2E(t *testing.T) {
	t.Log("Loading configuration...")
	var cfg E2ESlackConfig
	err := envconfig.Init(&cfg)
	require.NoError(t, err)

	cfg.Slack.Tester.CloudBasedTestEnabled = false // override property used only in the Cloud Slack E2E tests

	authHeaderValue := ""
	helmChartUninstalled := false
	gqlEndpoint := fmt.Sprintf("%s/%s", cfg.BotkubeCloud.APIBaseURL, cfg.BotkubeCloud.APIGraphQLEndpoint)

	if cfg.ScreenshotsDir != "" {
		t.Logf("Screenshots enabled. They will be saved to %s", cfg.ScreenshotsDir)
		err = os.MkdirAll(cfg.ScreenshotsDir, os.ModePerm)
		require.NoError(t, err)
	} else {
		t.Log("Screenshots disabled.")
	}

	t.Run("Connecting app", func(t *testing.T) {
		t.Log("Setting up browser...")

		launcher := launcher.New()
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

		var slackAppInstallationURL string
		if cfg.BotkubeCloud.APISlackAppInstallationBaseURLOverride == "" {
			slackAppInstallationURL = fmt.Sprintf("%s/%s", cfg.BotkubeCloud.APIBaseURL, cfg.BotkubeCloud.APISlackAppInstallationEndpoint)
		} else {
			slackAppInstallationURL = fmt.Sprintf("%s/%s", cfg.BotkubeCloud.APISlackAppInstallationBaseURLOverride, cfg.BotkubeCloud.APISlackAppInstallationEndpoint)
		}
		page.MustNavigate(slackAppInstallationURL).MustWaitStable()
		screenshotIfShould(t, cfg, page)

		isNgrok := strings.Contains(slackAppInstallationURL, "ngrok")
		if isNgrok {
			t.Log("ngrok host detected. Skipping the warning page...")
			page.MustElement("button.ant-btn").MustClick()
			screenshotIfShould(t, cfg, page)
		}

		t.Log("Logging in to Slack...")
		page.MustElement("input#domain").MustInput(cfg.Slack.WorkspaceName)
		screenshotIfShould(t, cfg, page)
		page.MustElementR("button", "Continue").MustClick()
		screenshotIfShould(t, cfg, page)
		page.MustElementR("a", "sign in with a password instead").MustClick()
		screenshotIfShould(t, cfg, page)
		page.MustElement("input#email").MustInput(cfg.Slack.Email)
		page.MustElement("input#password").MustInput(cfg.Slack.Password)
		screenshotIfShould(t, cfg, page)
		page.MustElementR("button", "(^Sign in$)|(^Sign In$)").MustClick()
		screenshotIfShould(t, cfg, page)

		t.Log("Installing Slack app...")
		time.Sleep(cfg.DefaultWaitTime) // ensure the screenshots shows a page after "Sign in" click
		screenshotIfShould(t, cfg, page)
		page.MustElementR("button.c-button:not(.c-button--disabled)", "Allow").MustClick()
		screenshotIfShould(t, cfg, page)
		page.MustElementR("a", "open this link in your browser")
		page.MustClose()

		t.Log("Opening new window...")
		// Workaround for the Slack protocol handler modal which cannot be closed programmatically
		slackPage := newBrowserPage(t, browser, cfg)
		t.Cleanup(func() {
			closePage(t, "slackPage", slackPage)
		})

		t.Logf("Navigating to the conversation with %q bot...", cfg.Slack.BotDisplayName)
		slackPage.MustNavigate(cfg.Slack.ConversationWithBotURL).MustWaitLoad()
		screenshotIfShould(t, cfg, slackPage)

		// sometimes it shows up - not sure if that really helps as I didn't see it later ¯\_(ツ)_/¯ We need to test it
		shortTimeoutPage := slackPage.Timeout(cfg.DefaultWaitTime)
		t.Cleanup(func() {
			closePage(t, "shortTimeoutPage", shortTimeoutPage)
		})
		elem, _ := shortTimeoutPage.Element("button.p-download_modal__not_now")
		if elem != nil {
			t.Log("Closing the 'Download the Slack app' modal...")
			elem.MustClick()
			time.Sleep(cfg.DefaultWaitTime) // to ensure the additional screenshot we do below shows closed modal
			screenshotIfShould(t, cfg, slackPage)
		}

		t.Log("Selecting the conversation with the bot...")
		screenshotIfShould(t, cfg, slackPage)
		slackPage.MustElementR(".c-scrollbar__child .c-virtual_list__scroll_container .p-channel_sidebar__static_list__item .p-channel_sidebar__name span", fmt.Sprintf("^%s$", cfg.Slack.BotDisplayName)).MustParent().MustParent().MustParent().MustClick()
		screenshotIfShould(t, cfg, slackPage)

		// it shows a popup for the new slack UI
		elem, _ = shortTimeoutPage.ElementR("button.c-button", "I’ll Explore on My Own")
		if elem != nil {
			t.Log("Closing the 'New Slack UI' modal...")
			elem.MustClick()
			time.Sleep(cfg.DefaultWaitTime) // to ensure the additional screenshot we do below shows closed modal
			screenshotIfShould(t, cfg, slackPage)
		}

		t.Log("Clicking 'Connect' button...")
		slackPage.MustElement(".p-actions_block__action button.c-button") // workaround for `MustElements` not having built-in retry
		screenshotIfShould(t, cfg, slackPage)
		elems := slackPage.MustElements(`.p-actions_block__action button.c-button`)
		require.NotEmpty(t, elems)
		t.Logf("Got %d buttons, using the last one...", len(elems))
		wait := slackPage.MustWaitOpen()
		elems[len(elems)-1].MustClick()
		botkubePage := wait()
		t.Cleanup(func() {
			closePage(t, "botkubePage", botkubePage)
		})

		t.Logf("Signing in to Botkube Cloud as %q...", cfg.BotkubeCloud.Email)
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElement("input#username").MustInput(cfg.BotkubeCloud.Email)
		botkubePage.MustElement("input#password").MustInput(cfg.BotkubeCloud.Password)
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElementR("form button[name='action'][data-action-button-primary='true']", "Continue").MustClick()
		screenshotIfShould(t, cfg, botkubePage)

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

		t.Log("Ensuring proper organizaton is selected")
		botkubePage.MustWaitOpen()
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElement("a.logo-link")

		pageURL := botkubePage.MustInfo().URL
		urlWithOrgID := appendOrgIDQueryParam(t, pageURL, cfg.BotkubeCloud.TeamOrganizationID)

		botkubePage.MustNavigate(urlWithOrgID).MustWaitLoad()
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElement("a.logo-link")
		screenshotIfShould(t, cfg, botkubePage)

		t.Log("Finalizing Slack workspace connection...")
		if cfg.Slack.WorkspaceAlreadyConnected {
			t.Log("Expecting already connected message...")
			botkubePage.MustElementR("div.ant-result-title", "Organization Already Connected!")
			return
		}

		t.Log("Finalizing connection...")
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElement("a.logo-link")
		screenshotIfShould(t, cfg, botkubePage)
		botkubePage.MustElementR("button > span", "Connect").MustParent().MustClick()
		screenshotIfShould(t, cfg, botkubePage)

		t.Log("Detecting homepage...")
		time.Sleep(cfg.DefaultWaitTime) // ensure the screenshots shows a view after button click
		screenshotIfShould(t, cfg, botkubePage)

		// Case 1: There are other instances on the list
		shortBkTimeoutPage := botkubePage.Timeout(cfg.DefaultWaitTime)
		t.Cleanup(func() {
			closePage(t, "shortBkTimeoutPage", shortBkTimeoutPage)
		})
		_, err := shortBkTimeoutPage.ElementR(".ant-layout-content p", "All Botkube installations managed by Botkube Cloud.")
		if err != nil {
			t.Logf("Failed to detect homepage with other instances created: %v", err)
			// Case 2:
			t.Logf("Checking if the homepage is in the 'no instances' state...")
			_, err := botkubePage.ElementR(".ant-layout-content h2", "Create your Botkube instance!")
			assert.NoError(t, err)
		}
	})

	t.Run("Run E2E tests with deployment", func(t *testing.T) {
		require.NotEmpty(t, authHeaderValue, "Previous subtest needs to pass to get authorization header value")

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
			gqlCli.MustDeleteSlackWorkspace(t, cfg.BotkubeCloud.TeamOrganizationID, slackWorkspace.ID)
		})

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

		t.Log("Creating deployment...")
		deployment := gqlCli.MustCreateBasicDeploymentWithCloudSlack(t, channel.Name(), slackWorkspace.TeamID, channel.Name())
		t.Cleanup(func() {
			// We have a glitch on backend side and the logic below is a workaround for that.
			// Tl;dr uninstalling Helm chart reports "DISCONNECTED" status, and deplyment deletion reports "DELETED" status.
			// If we do these two things too quickly, we'll run into resource version mismatch in repository logic.
			// Read more here: https://github.com/kubeshop/botkube-cloud/pull/486#issuecomment-1604333794

			for !helmChartUninstalled {
				t.Log("Waiting for Helm chart uninstallation, in order to proceed with deleting the first deployment...")
				time.Sleep(1 * time.Second)
			}

			t.Log("Helm chart uninstalled. Waiting a bit...")
			time.Sleep(3 * time.Second) // ugly, but at least we will be pretty sure we won't run into the resource version mismatch

			t.Log("Deleting first deployment...")
			gqlCli.MustDeleteDeployment(t, graphql.ID(deployment.ID))
		})

		t.Log("Creating a second deployment...")
		deployment2 := gqlCli.MustCreateBasicDeploymentWithCloudSlack(t, fmt.Sprintf("%s-2", channel.Name()), slackWorkspace.TeamID, channel.Name())
		t.Cleanup(func() {
			t.Log("Deleting second deployment...")
			gqlCli.MustDeleteDeployment(t, graphql.ID(deployment2.ID))
		})

		params := helmx.InstallChartParams{
			RepoURL:       "https://storage.googleapis.com/botkube-latest-main-charts",
			RepoName:      "botkube",
			Name:          "botkube",
			Namespace:     "botkube",
			Command:       *deployment.HelmCommand,
			PluginRepoURL: cfg.BotkubeCloud.PluginRepoURL,
		}
		helmInstallCallback := helmx.InstallChart(t, params)
		t.Cleanup(func() {
			t.Log("Uninstalling Helm chart...")
			helmInstallCallback(t)
			helmChartUninstalled = true
		})

		t.Log("Waiting for help message...")
		assertionFn := func(msg string) (bool, int, string) {
			return strings.Contains(msg, fmt.Sprintf("Botkube instance %q is now active.", deployment.Name)), 0, ""
		}
		err = tester.WaitForMessagePosted(tester.BotUserID(), channel.ID(), 3, assertionFn)

		cmdHeader := func(command string) string {
			return fmt.Sprintf("`%s` on `%s`", command, deployment.Name)
		}

		t.Run("Check basic commands", func(t *testing.T) {
			t.Log("Testing ping with --cluster-name")
			command := fmt.Sprintf("ping --cluster-name %s", deployment.Name)
			expectedMessage := fmt.Sprintf("`%s` on `%s`\n```\npong", command, deployment.Name)
			tester.PostMessageToBot(t, channel.Identifier(), command)
			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)

			t.Log("Testing ping for not connected deployment #2")
			command = "ping"
			expectedMessage = fmt.Sprintf("The cluster %s (id: %s) is not connected.", deployment2.Name, deployment2.ID)
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("%s --cluster-name %s", command, deployment2.Name))

			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			if err != nil { // the new cloud backend not release yet
				t.Logf("Fallback to the old behavior with message sent at the channel level...")
				err = tester.OnChannel().WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			}
			require.NoError(t, err)

			t.Log("Testing ping for not existing deployment")
			command = "ping"
			deployName := "non-existing-deployment"
			expectedMessage = fmt.Sprintf("*Instance not found* The cluster %q does not exist.", deployName)
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("%s --cluster-name %s", command, deployName))
			err = tester.WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			if err != nil { // the new cloud backend not release yet
				t.Logf("Fallback to the old behavior with message sent at the channel level...")
				err = tester.OnChannel().WaitForLastMessageContains(tester.BotUserID(), channel.ID(), expectedMessage)
			}
			require.NoError(t, err)

			t.Log("Setting cluster as default")
			tester.PostMessageToBot(t, channel.Identifier(), fmt.Sprintf("cloud set default-instance %s", deployment.ID))
			t.Log("Waiting for confirmation message...")
			expectedClusterDefaultMsg := fmt.Sprintf(":white_check_mark: Instance %s was successfully selected as the default cluster for this channel.", deployment.Name)
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedClusterDefaultMsg)
			if err != nil { // the new cloud backend not release yet
				t.Logf("Fallback to the old behavior with message sent at the channel level...")
				err = tester.OnChannel().WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedClusterDefaultMsg)
			}
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
					fmt.Sprintf("*Cluster:* %s", deployment.Name),
					"*Recommendations*",
					fmt.Sprintf("Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name),
					fmt.Sprintf("The 'latest' tag used in 'nginx:latest' image of Pod '%s/%s' container 'nginx' should be avoided.", pod.Namespace, pod.Name),
				}

				var result = true
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
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", deployment.Name)

			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log("Waiting for watch begin message...")
			expectedWatchBeginMsg := fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", deployment.Name)
			recentMessages := 2 // take into the account the optional "upgrade checker message"
			err = tester.OnChannel().WaitForMessagePosted(tester.BotUserID(), channel.ID(), recentMessages, func(msg string) (bool, int, string) {
				if !strings.EqualFold(expectedWatchBeginMsg, msg) {
					count := diff.CountMatchBlock(expectedWatchBeginMsg, msg)
					msgDiff := diff.Diff(expectedWatchBeginMsg, msg)
					return false, count, msgDiff
				}
				return true, 0, ""
			})

			t.Log("Verifying disabled notification on Cloud...")
			deploy := gqlCli.MustGetDeployment(t, graphql.ID(deployment.ID))
			require.True(t, *deploy.Platforms.CloudSlacks[0].Channels[0].NotificationsDisabled)

			t.Log("Verifying disabled notifications on chat...")
			command := "status notifications"
			expectedBody := formatx.CodeBlock(fmt.Sprintf("Notifications from cluster '%s' are disabled here.", deployment.Name))
			expectedMessage := fmt.Sprintf("%s\n%s", cmdHeader(command), expectedBody)
			tester.PostMessageToBot(t, channel.Identifier(), "status notifications")
			err = tester.WaitForLastMessageEqual(tester.BotUserID(), channel.ID(), expectedMessage)
			require.NoError(t, err)
		})

		t.Run("Cloud -> Botkube Deployment sync", func(t *testing.T) {
			t.Log("Removing source binding from Slack platform & add actions")
			d := gqlCli.MustGetDeployment(t, graphql.ID(deployment.ID)) // Get final resource version
			deployment = removeSourcesAndAddActions(t, gqlCli.Client, &d)

			t.Log("Waiting for config reload message...")
			expectedReloadMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", deployment.Name)
			err = tester.OnChannel().WaitForMessagePostedRecentlyEqual(tester.BotUserID(), channel.ID(), expectedReloadMsg)
			require.NoError(t, err)

			t.Log("Waiting for watch begin message...")
			expectedWatchBeginMsg := fmt.Sprintf("My watch begins for cluster '%s'! :crossed_swords:", deployment.Name)
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
					DeploymentID: &deployment.ID,
				},
			}

			err := gqlCli.Query(context.Background(), &auditPage, variables)
			require.NoError(t, err)
			require.NotEmpty(t, auditPage.Audits.Data)

			t.Log("Asserting command executed events...")
			botPlatform := gqlModel.BotPlatformSLACk
			want := ExpectedCommandExecutedEvents([]string{
				"kubectl get deployments -A",
				fmt.Sprintf("ping --cluster-name %s", deployment.Name),
				"disable notifications",
				"status notifications",
				"list sources",
				"list actions",
			}, &botPlatform, channel.Name())

			got := CommandExecutedEventsFromAuditResponse(auditPage.Audits)
			require.ElementsMatch(t, want, got)

			t.Log("Asserting source emitted events...")
			wantSrcEvents := []gqlModel.SourceEventEmittedEvent{
				{
					Source: &gqlModel.SourceEventDetails{
						Name:        "kubernetes_config",
						DisplayName: "Kubernetes Info",
					},
					PluginName: "botkube/kubernetes",
				},
			}
			gotSrcEvents := SourceEmittedEventsFromAuditResponse(auditPage.Audits)
			require.ElementsMatch(t, wantSrcEvents, gotSrcEvents)
		})
	})
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
		var channelUpdateInputs []*gqlModel.ChannelBindingsByNameUpdateInput
		for _, channel := range slack.Channels {
			channelUpdateInputs = append(channelUpdateInputs, &gqlModel.ChannelBindingsByNameUpdateInput{
				Name: channel.Name,
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
			Actions:         CreateActionUpdateInput(),
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
	data, err := page.Screenshot(false, nil)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	err = os.WriteFile(filePath, data, 0644)
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
