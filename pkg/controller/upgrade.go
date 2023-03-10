package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	defaultDuration = 24 * time.Hour
	upgradeMsgFmt   = "Newer version (%s) of Botkube is available :tada:. Please upgrade Botkube backend.\nVisit botkube.io for more info."
	repoOwner       = "kubeshop"
	repoName        = "botkube"
)

// GitHubRepoClient describes the client for getting latest release for a given repository.
type GitHubRepoClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// UpgradeChecker checks for new Botkube releases.
type UpgradeChecker struct {
	log       logrus.FieldLogger
	notifiers []notifier.Bot
	ghRepoCli GitHubRepoClient
}

// NewUpgradeChecker creates a new instance of the Upgrade Checker.
func NewUpgradeChecker(log logrus.FieldLogger, notifiers []notifier.Bot, ghCli GitHubRepoClient) *UpgradeChecker {
	return &UpgradeChecker{log: log, notifiers: notifiers, ghRepoCli: ghCli}
}

// Run runs the Upgrade Checker and checks for new Botkube releases periodically.
func (c *UpgradeChecker) Run(ctx context.Context) error {
	c.log.Info("Starting checker")
	// Check at startup
	notified, err := c.notifyAboutUpgradeIfShould(ctx)
	if err != nil {
		return fmt.Errorf("while notifying about upgrade if should: %w", err)
	}

	if notified {
		return nil
	}

	ticker := time.NewTicker(defaultDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Shutdown requested. Finishing...")
			return nil
		case <-ticker.C:
			// Check periodically
			notified, err := c.notifyAboutUpgradeIfShould(ctx)
			if err != nil {
				wrappedErr := fmt.Errorf("while notifying about upgrade if should: %w", err)
				c.log.Error(wrappedErr.Error())
			}

			if notified {
				return nil
			}
		}
	}
}

func (c *UpgradeChecker) notifyAboutUpgradeIfShould(ctx context.Context) (bool, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, repoOwner, repoName)
	if err != nil {
		return false, fmt.Errorf("while getting latest release from GitHub: %w", err)
	}

	if release == nil || release.TagName == nil {
		return false, errors.New("release tag is empty")
	}

	c.log.Debugf("latest release tag: %s", *release.TagName)

	// Send notification if newer version available
	if version.Short() == *release.TagName {
		// no new version, finish
		return false, nil
	}

	err = notifier.SendPlaintextMessage(ctx, c.notifiers, fmt.Sprintf(upgradeMsgFmt, *release.TagName))
	if err != nil {
		return false, fmt.Errorf("while sending message about new release: %w", err)
	}

	c.log.Infof("Notified about new release %q. Finishing...", *release.TagName)
	return true, nil
}
