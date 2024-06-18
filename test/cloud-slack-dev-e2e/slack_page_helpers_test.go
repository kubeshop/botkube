//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"testing"
	"time"

	"github.com/go-rod/rod"
)

const slackBaseURL = "slack.com"

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
		p.page.Screenshot()
		p.page.MustElement("button#slack-workspace-connect").MustClick().
			MustWaitEnabled() // when it's re-enabled, then it means the query was finished 
		p.page.Screenshot()
	}

	_ = p.page.Close() // the page should be closed automatically anyway
}
