package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v27/github"
	"github.com/infracloudio/botkube/pkg/config"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/notify"
)

var (
	notified          = false
	botkubeUpgradeMsg = "Newer version (%s) of BotKube is available :tada:. Please upgrade BotKube backend.\n" +
		"Visit botkube.io for more info."
)

func checkRelease(c *config.Config, notifiers []notify.Notifier) {
	ctx := context.Background()
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "infracloudio", "botkube")
	if err == nil {
		log.Logger.Debugf(fmt.Sprintf("Upgrade notifier:: latest release info=%+v", release))
		if len(os.Getenv("BOTKUBE_VERSION")) == 0 || release.TagName == nil {
			return
		}

		// Send notification if newer version available
		if len(os.Getenv("BOTKUBE_VERSION")) > 0 && os.Getenv("BOTKUBE_VERSION") != *release.TagName {
			sendMessage(c, notifiers, fmt.Sprintf(botkubeUpgradeMsg, *release.TagName))
			notified = true
		}
	}
}

// UpgradeNotifier checks if newer version for BotKube backend available and notifies user
func UpgradeNotifier(c *config.Config, notifiers []notify.Notifier) {
	// Check at startup
	checkRelease(c, notifiers)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if notified == true {
				return
			}
			// Check periodically
			checkRelease(c, notifiers)
		}
	}
}
