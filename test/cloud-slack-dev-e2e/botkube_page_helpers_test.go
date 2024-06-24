//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"botkube.io/botube/test/cloud_graphql"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/require"

	gqlModel "github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/graphql"
)

const (
	authHeaderName            = "Authorization"
	awaitInstanceStatusChange = 2 * time.Minute
	orgQueryParam             = "organizationId"
)

type BotkubeCloudPage struct {
	cfg  E2ESlackConfig
	page *Page

	AuthHeaderValue string
	GQLEndpoint     string
	ConnectedDeploy *gqlModel.Deployment
}

func NewBotkubeCloudPage(t *testing.T, cfg E2ESlackConfig) *BotkubeCloudPage {
	return &BotkubeCloudPage{
		page:        &Page{t: t, cfg: cfg},
		cfg:         cfg,
		GQLEndpoint: fmt.Sprintf("%s/%s", cfg.BotkubeCloud.APIBaseURL, cfg.BotkubeCloud.APIGraphQLEndpoint),
	}
}

func (p *BotkubeCloudPage) NavigateAndLogin(t *testing.T, page *rod.Page) {
	t.Log("Log into Botkube Cloud Dashboard")

	p.page.Page = page

	p.page.MustNavigate(appendOrgIDQueryParam(t, p.cfg.BotkubeCloud.UIBaseURL, p.cfg.BotkubeCloud.TeamOrganizationID))
	p.page.MustWaitNavigation()

	p.page.MustElement(`input[name="username"]`).MustInput(p.cfg.BotkubeCloud.Email)
	p.page.MustElement(`input[name="password"]`).MustInput(p.cfg.BotkubeCloud.Password)
	p.page.MustElementR("button", "^Continue$").MustClick()
	p.page.Screenshot()
}

func (p *BotkubeCloudPage) HideCookieBanner(t *testing.T) {
	t.Log("Hide Botkube cookie banner")
	p.page.MustElementR("button", "^Decline$").MustClick()
	p.page.Screenshot()
}

func (p *BotkubeCloudPage) InterceptBearerToken(t *testing.T, browser *rod.Browser) func() {
	t.Logf("Starting hijacking requests to %q to get the bearer token...", p.GQLEndpoint)

	router := browser.HijackRequests()
	router.MustAdd(p.GQLEndpoint, func(ctx *rod.Hijack) {
		if p.AuthHeaderValue != "" {
			ctx.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		if ctx.Request != nil && ctx.Request.Method() != http.MethodPost {
			ctx.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		require.NotNil(t, ctx.Request)
		p.AuthHeaderValue = ctx.Request.Header(authHeaderName)
		t.Log("Bearer token intercepted")
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()
	return router.MustStop
}

func (p *BotkubeCloudPage) CreateNewInstance(t *testing.T, name string) {
	t.Log("Create new Botkube Instance")

	p.page.MustElement("h6#create-instance").MustClick()
	time.Sleep(3 * time.Second)
	p.page.Screenshot("after-clicking-create-instance")
	p.page.MustElement(`input[name="name"]`).MustSelectAllText().MustInput(name)
	p.page.Screenshot("after-filling-in-instance-name")

	// persist connected deploy info
	_, id, _ := strings.Cut(p.page.MustInfo().URL, "add/")
	p.ConnectedDeploy = &gqlModel.Deployment{
		Name: name,
		ID:   id,
	}
}

func (p *BotkubeCloudPage) InstallAgentInCluster(t *testing.T, botkubeBinary string) {
	t.Log("Getting Botkube install command")
	installCmd := p.page.MustElement("div#install-upgrade-cmd > kbd").MustText()

	t.Log("Installing Botkube using Botkube CLI")
	args, err := shellwords.Parse(installCmd)
	args = append(args, "--auto-approve")
	require.NoError(t, err)

	cmd := exec.Command(botkubeBinary, args[1:]...)
	installOutput, err := cmd.CombinedOutput()
	t.Log(string(installOutput))
	require.NoError(t, err)

	p.page.MustElement("button#cluster-connected").MustClick()
}

func (p *BotkubeCloudPage) OpenSlackAppIntegrationPage(t *testing.T) {
	t.Log("Opening Slack App Integration Page")
	p.page.MustElement(`button[aria-label="Add tab"]`).MustClick()
	p.page.MustWaitStable()
	p.page.MustElementR("button", "^Slack$").MustClick()
	p.page.MustWaitStable()
	p.page.Screenshot()

	p.page.MustElementR("a", "Add to Slack").MustClick()
}

// ReAddSlackPlatformIfShould add the slack platform again as the page was often not refreshed with a newly connected Slack Workspace.
// It only occurs with headless mode.
func (p *BotkubeCloudPage) ReAddSlackPlatformIfShould(t *testing.T, isHeadless bool) {
	if !isHeadless {
		return
	}

	t.Log("Re-adding Slack platform")

	p.page.MustActivate()
	p.page.MustElement(`button[aria-label="remove"]`).MustClick()
	p.page.MustElement(`button[aria-label="Add tab"]`).MustClick()
	p.page.MustElementR("button", "^Slack$").MustClick()
	p.page.Screenshot()
}

func (p *BotkubeCloudPage) VerifyDeploymentStatus(t *testing.T, status string) {
	t.Logf("Waiting for status '%s'", status)
	p.page.Timeout(awaitInstanceStatusChange).MustElementR("div#deployment-status", status)
}

func (p *BotkubeCloudPage) SetupSlackWorkspace(t *testing.T, channel string) {
	t.Logf("Selecting newly connected %q Slack Workspace", p.cfg.Slack.WorkspaceName)

	p.page.MustElement(`input[type="search"]`).
		MustInput(p.cfg.Slack.WorkspaceName).
		MustType(input.Enter)
	p.page.Screenshot()

	// filter by channel, to make sure that it's visible on the first table page, in order to select it in the next step
	t.Log("Filtering by channel name")
	p.page.Mouse.MustScroll(10, 5000) // scroll bottom, as the footer collides with selecting filter
	p.page.Screenshot()
	p.page.MustElement("table th:nth-child(3) span.ant-dropdown-trigger.ant-table-filter-trigger").MustClick()

	t.Log("Selecting channel checkbox")
	p.page.MustElement("input#name-channel").MustInput(channel).MustType(input.Enter)
	p.page.MustElement(fmt.Sprintf(`input[type="checkbox"][name="%s"]`, channel)).MustClick()
	p.page.Screenshot()
}

func (p *BotkubeCloudPage) FinishWizard(t *testing.T) {
	t.Log("Navigating to plugin selection")
	p.page.Screenshot("before-first-next")

	time.Sleep(3 * time.Second)
	p.page.MustElementR("button", "/^Next$/i").
		MustWaitEnabled().
		// We need to wait, otherwise, we click the same 'Next' button twice before the query is executed, and we are not really
		// moved to the next step. Updating the navigation would resolve that issue.
		MustClick().MustWaitStable()

	p.page.Screenshot("after-first-next")

	t.Log("Using pre-selected plugins. Navigating to wizard summary")
	time.Sleep(3 * time.Second)
	p.page.MustElementR("button", "/^Next$/i").
		MustWaitEnabled().
		// We need to wait, otherwise, we click the same 'Next' button twice before the query is executed, and we are not really
		// moved to the next step. Updating the navigation would resolve that issue.
		MustClick().MustWaitStable()
	p.page.Screenshot("after-second-next")

	t.Log("Submitting changes")
	p.page.Mouse.MustMoveTo(0, 0)
	time.Sleep(3 * time.Second)
	p.page.MustElementR("button", "/^Deploy changes$/i").
		MustWaitEnabled().
		MustClick()
	p.page.Screenshot("after-deploy-changes")

	// wait till gql mutation passes, and navigates to instance details, otherwise, we could navigate to instance details with state 'draft'
	p.page.MustWaitNavigation()
	p.page.Screenshot("after-deploy-changes-navigation")
}

func (p *BotkubeCloudPage) UpdateKubectlNamespace(t *testing.T) {
	t.Log("Updating 'kubectl' namespace property")

	p.openKubectlUpdateForm()

	p.page.MustElementR("input#root_defaultNamespace", "default").MustSelectAllText().MustInput("kube-system")
	p.page.Screenshot("after-changing-namespace-property")
	p.page.MustElementR("button", "/^Update$/i").MustClick()
	p.page.Screenshot("after-clicking-plugin-update")

	t.Log("Moving to top left corner of the page")
	p.page.Mouse.MustMoveTo(0, 0)
	time.Sleep(3 * time.Second)

	t.Log("Submitting changes")
	p.page.MustWaitStable()
	p.page.Screenshot("before-deploying-plugin-changes")
	p.page.MustElementR("button", "/Deploy changes/i").MustClick()
	p.page.Screenshot("after-deploying-plugin-changes")
}

func (p *BotkubeCloudPage) VerifyUpdatedKubectlNamespace(t *testing.T) {
	t.Log("Verifying that the 'namespace' value was updated and persisted properly")

	p.openKubectlUpdateForm()
	p.page.MustElementR("input#root_defaultNamespace", "kube-system")
}

func (p *BotkubeCloudPage) openKubectlUpdateForm() {
	p.page.Screenshot("before-selecting-plugins-tab")
	p.page.MustElementR(`div[role="tab"]`, "Plugins").MustFocus().MustClick().MustWaitStable()

	p.page.MustWaitStable()
	p.page.Screenshot("after-selecting-plugins-tab")

	p.page.MustElement(`button[id^="botkube/kubectl_"]`).
		MustWaitEnabled(). // needed as we have an "Outdated version detected" glitch
		MustClick()
	p.page.Screenshot("after-opening-kubectl-cfg")

	p.page.MustElement(`div[data-node-key="ui-form"]`).MustClick()
	p.page.Screenshot("after-selecting-kubectl-cfg-form")
}

func (p *BotkubeCloudPage) CleanupOnFail(t *testing.T, gqlCli *cloud_graphql.Client) {
	t.Log("Cleaning up Botkube instance on test failure...")

	if p.ConnectedDeploy == nil {
		t.Log("No deployment to delete")
		return
	}

	deleteDeployment(t, gqlCli, p.ConnectedDeploy.ID, "connected")
}

func appendOrgIDQueryParam(t *testing.T, inURL, orgID string) string {
	parsedURL, err := url.Parse(inURL)
	require.NoError(t, err)
	queryValues := parsedURL.Query()
	queryValues.Set(orgQueryParam, orgID)
	parsedURL.RawQuery = queryValues.Encode()

	return parsedURL.String()
}
