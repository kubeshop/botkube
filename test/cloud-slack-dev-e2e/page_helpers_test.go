//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Chromium is not supported by Slack web app for some reason
	// Currently, we get:
	//   This browser won’t be supported starting September 1st, 2024. Update your browser to keep using Slack. Learn more:
	//   https://slack.com/intl/en-gb/help/articles/1500001836081-Slack-support-life-cycle-for-operating-systems-app-versions-and-browsers
	chromeUserAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

type Page struct {
	*rod.Page
	t   *testing.T
	cfg E2ESlackConfig
}

func (p Page) Screenshot() {
	p.t.Helper()
	if p.cfg.ScreenshotsDir == "" {
		return
	}

	pathParts := strings.Split(p.cfg.ScreenshotsDir, "/")
	pathParts = append(pathParts)

	filePath := filepath.Join(p.cfg.ScreenshotsDir, fmt.Sprintf("%d.png", time.Now().UnixNano()))

	logMsg := fmt.Sprintf("Saving screenshot to %q", filePath)
	if p.cfg.DebugMode {
		info, err := p.Info()
		assert.NoError(p.t, err)

		if info != nil {
			logMsg += fmt.Sprintf(" for URL %q", info.URL)
		}
	}
	p.t.Log(logMsg)
	data, err := p.Page.Screenshot(false, nil)
	assert.NoError(p.t, err)
	if err != nil {
		return
	}

	err = os.WriteFile(filePath, data, 0o644)
	assert.NoError(p.t, err)
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

