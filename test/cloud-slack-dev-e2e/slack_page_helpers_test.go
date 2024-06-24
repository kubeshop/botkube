//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"botkube.io/botube/test/cloud_graphql"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

const (
	slackBaseURL          = "slack.com"
	waitTime              = 10 * time.Second
	contextTimeout        = 30 * time.Second
	shorterContextTimeout = 10 * time.Second
)

type SlackPage struct {
	page *Page
	cfg  E2ESlackConfig
}

func NewSlackPage(t *testing.T, cfg E2ESlackConfig) *SlackPage {
	return &SlackPage{
		page: &Page{t: t, cfg: cfg},
		cfg:  cfg,
	}
}

func (p *SlackPage) ConnectWorkspace(t *testing.T, browser *rod.Browser) {
	p.page.Page = browser.MustPages().MustFindByURL(slackBaseURL)
	p.page.MustWaitStable()

	defer func(page *Page) {
		err := page.Close()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			t.Fatalf("Failed to close page: %s", err.Error())
		}
	}(p.page)

	p.page.MustElement("input#domain").MustInput(p.cfg.Slack.WorkspaceName)
	p.page.MustElementR("button", "Continue").MustClick()
	p.page.Screenshot("after-continue")

	p.page.MustWaitStable()
	p.page.MustElementR("a", "sign in with a password instead").MustClick()
	p.page.Screenshot("after-sign-in-with-password")
	p.page.MustElement("input#email").MustInput(p.cfg.Slack.Email)
	p.page.MustElement("input#password").MustInput(p.cfg.Slack.Password)
	p.page.Screenshot()

	t.Log("Hide Slack cookie banner that collides with 'Sign in' button")
	pageWithTimeout := p.page.Timeout(shorterContextTimeout)
	t.Cleanup(func() {
		_ = pageWithTimeout.Close()
	})

	cookieElem, err := pageWithTimeout.Element("button#onetrust-accept-btn-handler")
	if err != nil {
		t.Logf("Failed to obtain cookie element: %s. Skipping...", err.Error())
	} else {
		cookieElem.MustClick()
	}

	p.page.MustElementR("button", "/^Sign in$/i").MustClick()
	p.page.Screenshot("after-sign-in")

	time.Sleep(waitTime) // ensure the screenshots shows a page after "Sign in" click
	p.page.Screenshot("after-sign-in-page")
	p.page.MustElementR("button.c-button:not(.c-button--disabled)", "Allow").MustClick()

	t.Log("Finalizing Slack workspace connection...")
	if p.cfg.Slack.WorkspaceAlreadyConnected {
		t.Log("Expecting already connected message...")
		p.page.MustElementR("div.ant-result-title", "Organization Already Connected!")
	} else {
		t.Log("Finalizing connection...")
		time.Sleep(waitTime)
		p.page.Screenshot("before-workspace-connect")
		p.page.MustElement("button#slack-workspace-connect").MustClick()
		p.page.Screenshot("after-workspace-connect")
	}

	t.Log("Waiting for page auto-close...")
	err = p.page.WaitIdle(waitTime) // wait for auto-close
	assert.NoError(t, err)
}

func (p *SlackPage) CleanupOnFail(t *testing.T, gqlCli *cloud_graphql.Client) {
	t.Log("Cleaning up Slack workspace on test failure...")
	if !p.cfg.Slack.DisconnectWorkspaceAfterTests {
		return
	}

	if !t.Failed() {
		t.Log("Skipping Slack workspace cleanup as the subtest passed...")
		return
	}

	slackWorkspace := findConnectedSlackWorkspace(t, p.cfg, gqlCli)
	disconnectConnectedSlackWorkspace(t, p.cfg, gqlCli, slackWorkspace)
}
