package reloader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/r3labs/diff/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/config"
)

// DeploymentClient defines GraphQL client.
type DeploymentClient interface {
	GetConfigWithResourceVersion(ctx context.Context) (remote.Deployment, error)
}

// NewRemote returns new ConfigUpdater.
func NewRemote(log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, restarter *Restarter, resVerHolders ...ResourceVersionHolder) *RemoteConfigReloader {
	return &RemoteConfigReloader{
		log:           log,
		interval:      interval,
		deployCli:     deployCli,
		resVerHolders: resVerHolders,
		restarter:     restarter,
	}
}

type RemoteConfigReloader struct {
	log           logrus.FieldLogger
	interval      time.Duration
	resVerHolders []ResourceVersionHolder

	latestCfg  config.Config
	resVersion int

	deployCli DeploymentClient
	restarter *Restarter
}

func (u *RemoteConfigReloader) Do(ctx context.Context) error {
	u.log.Info("Starting...")

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			u.log.Info("Shutdown requested. Finishing...")
			return nil
		case <-ticker.C:
			u.log.Debug("Querying the latest configuration...")
			// Check periodically
			cfg, resVer, err := u.queryConfig(ctx)
			if err != nil {
				wrappedErr := fmt.Errorf("while getting latest config: %w", err)
				u.log.Error(wrappedErr.Error())
				continue
			}
			cfgDiff, err := u.processNewConfig(cfg, resVer)
			if err != nil {
				wrappedErr := fmt.Errorf("while processing new config: %w", err)
				u.log.Error(wrappedErr.Error())
				continue
			}

			if !cfgDiff.shouldRestart {
				continue
			}

			u.log.Info("Reloading configuration...")
			err = u.restarter.Do(ctx)
			if err != nil {
				u.log.Errorf("while restarting the app: %s", err.Error())
				return fmt.Errorf("while restarting the app: %w", err)
			}
		}
	}
}

func (u *RemoteConfigReloader) queryConfig(ctx context.Context) (config.Config, int, error) {
	deploy, err := u.deployCli.GetConfigWithResourceVersion(ctx)
	if err != nil {
		return config.Config{}, 0, fmt.Errorf("while getting deployment: %w", err)
	}

	var latestCfg config.Config
	err = yaml.Unmarshal([]byte(deploy.YAMLConfig), &latestCfg)
	if err != nil {
		return config.Config{}, 0, fmt.Errorf("while unmarshaling config: %w", err)
	}

	return latestCfg, deploy.ResourceVersion, nil
}

type configDiff struct {
	shouldRestart bool
}

func (u *RemoteConfigReloader) processNewConfig(newCfg config.Config, newResVer int) (configDiff, error) {
	if newResVer == u.resVersion {
		u.log.Debugf("Config version (%d) is the same as the latest one. Skipping...", newResVer)
		return configDiff{}, nil
	}
	if newResVer < u.resVersion {
		return configDiff{}, fmt.Errorf("current config version (%d) is newer than the latest one (%d)", u.resVersion, newResVer)
	}
	u.setResourceVersionForAll(newResVer)

	changelog, err := diff.Diff(u.latestCfg, newCfg, diff.DisableStructValues(), diff.AllowTypeMismatch(true))
	if err != nil {
		return configDiff{}, fmt.Errorf("while diffing configs: %w", err)
	}

	if len(changelog) == 0 {
		u.log.Debugf("Config with higher version (%d) is the same as the latest one. No need to reload config", newResVer)
		return configDiff{}, nil
	}

	var paths []string
	for _, change := range changelog {
		paths = append(paths, fmt.Sprintf(`- "%s"`, strings.Join(change.Path, ".")))
	}
	u.log.Debugf("detected config changes on paths:\n%s", strings.Join(paths, "\n"))

	// TODO: check if notifications are enabled and if so:
	//  - update notifications for a given channel (this needs a global state)
	//  - send message to a given channel (this needs a rework for the notifier executor)
	//    - updating notifications should happen after ConfigMap update, not before
	//    - same for remote config reloader
	//  - do not restart the app

	u.latestCfg = newCfg
	u.log.Debugf("Successfully set newer config version (%d). Config should be reloaded soon", newResVer)
	return configDiff{
		shouldRestart: true,
	}, nil
}

func (u *RemoteConfigReloader) setResourceVersionForAll(resVersion int) {
	u.resVersion = resVersion
	for _, h := range u.resVerHolders {
		h.SetResourceVersion(u.resVersion)
	}
}
