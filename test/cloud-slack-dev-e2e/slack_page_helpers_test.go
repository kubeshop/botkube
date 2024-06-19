//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

const (
	slackBaseURL = "slack.com"
	waitTime     = 10 * time.Second
)

type SlackPage struct {
	page *Page
	cfg  SlackConfig
}

func NewSlackPage(t *testing.T, cfg E2ESlackConfig) *SlackPage {
	return &SlackPage{
		page: &Page{t: t, cfg: cfg},
		cfg:  cfg.Slack,
	}
}

func (p *SlackPage) ConnectWorkspace(t *testing.T, headless bool, browser *rod.Browser) {
	p.page.Page = browser.MustPages().MustFindByURL(slackBaseURL)

	p.page.MustElement("input#domain").MustInput(p.cfg.WorkspaceName)

	p.page.MustElementR("button", "Continue").MustClick()
	p.page.Screenshot()

	// here we get reloaded, so we need to type it again (looks like bug on Slack side)
	if !headless {
		p.page.MustElement("input#domain").MustInput(p.cfg.WorkspaceName)
		p.page.MustElementR("button", "Continue").MustClick()
	}

	p.page.MustWaitStable()
	p.page.MustElementR("a", "sign in with a password instead").MustClick()
	p.page.Screenshot()
	p.page.MustElement("input#email").MustInput(p.cfg.Email)
	p.page.MustElement("input#password").MustInput(p.cfg.Password)
	p.page.Screenshot()

	t.Log("Hide Slack cookie banner that collides with 'Sign in' button")
	cookie, err := p.page.Timeout(5 * time.Second).Element("button#onetrust-accept-btn-handler")
	if err != nil {
		t.Logf("Failed to obtain cookie element: %s. Skipping...", err.Error())
	} else {
		cookie.MustClick()
	}

	p.page.MustElementR("button", "/^Sign in$/i").MustClick()
	p.page.Screenshot()

	p.page.MustElementR("button.c-button:not(.c-button--disabled)", "Allow").MustClick()

	t.Log("Finalizing Slack workspace connection...")
	if p.cfg.WorkspaceAlreadyConnected {
		t.Log("Expecting already connected message...")
		p.page.MustElementR("div.ant-result-title", "Organization Already Connected!")
	} else {
		t.Log("Finalizing connection...")
		time.Sleep(3 * time.Second)
		p.page.Screenshot()
		p.page.MustElement("button#slack-workspace-connect").MustClick()
		p.page.Screenshot()
	}

	p.waitForHomepage(t)

	_ = p.page.Close() // the page should be closed automatically anyway
}

func (p *SlackPage) waitForHomepage(t *testing.T) {
	t.Log("Detecting homepage...")
	time.Sleep(waitTime) // ensure the screenshots shows a view after button click
	p.page.Screenshot()

	// Case 1: There are other instances on the list
	shortBkTimeoutPage := p.page.Timeout(waitTime)
	t.Cleanup(func() {
		closePage(t, "shortBkTimeoutPage", shortBkTimeoutPage)
	})
	_, err := shortBkTimeoutPage.ElementR(".ant-layout-content p", "All Botkube installations managed by Botkube Cloud.")
	if err != nil {
		t.Logf("Failed to detect homepage with other instances created: %v", err)
		// Fallback to Case 2: No other instances created
		t.Logf("Checking if the homepage is in the 'no instances' state...")
		_, err := p.page.ElementR(".ant-layout-content h2", "Create your Botkube instance!")
		assert.NoError(t, err)
	}
}
