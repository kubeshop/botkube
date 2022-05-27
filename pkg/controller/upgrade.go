// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/google/go-github/v44/github"

	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/version"
)

const (
	defaultDuration = 24 * time.Hour
	upgradeMsgFmt   = "Newer version (%s) of BotKube is available :tada:. Please upgrade BotKube backend.\nVisit botkube.io for more info."
	repoOwner       = "infracloudio"
	repoName        = "botkube"
)

// GitHubRepoClient describes the client for getting latest release for a given repository.
type GitHubRepoClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// UpgradeChecker checks for new BotKube releases.
type UpgradeChecker struct {
	log       logrus.FieldLogger
	notifiers []notify.Notifier
	ghRepoCli GitHubRepoClient
}

// NewUpgradeChecker creates a new instance of the Upgrade Checker.
func NewUpgradeChecker(log logrus.FieldLogger, notifiers []notify.Notifier, ghCli GitHubRepoClient) *UpgradeChecker {
	return &UpgradeChecker{log: log, notifiers: notifiers, ghRepoCli: ghCli}
}

// Run runs the Upgrade Checker and checks for new BotKube releases periodically.
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
			c.log.Info("Context canceled. Finishing...")
			return errors.New("test")
		case <-ticker.C:
			// Check periodically
			notified, err := c.notifyAboutUpgradeIfShould(ctx)
			if err != nil {
				return fmt.Errorf("while notifying about upgrade if should: %w", err)
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

	c.log.Debugf("latest release info: %+v", release)
	if release.TagName == nil {
		return false, errors.New("release tag is empty")
	}

	// Send notification if newer version available
	if version.Short() == *release.TagName {
		// no new version, finish
		return false, nil
	}

	err = sendMessageToNotifiers(ctx, c.notifiers, fmt.Sprintf(upgradeMsgFmt, *release.TagName))
	if err != nil {
		return false, fmt.Errorf("while sending message about new release: %w", err)
	}

	c.log.Infof("Notified about new release %q. Finishing...", *release.TagName)
	return true, nil
}
