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
	"github.com/kubeshop/botkube/internal/status"
	"github.com/kubeshop/botkube/pkg/config"
)

var _ Reloader = &RemoteConfigReloader{}

// DeploymentClient defines GraphQL client.
type DeploymentClient interface {
	GetResourceVersion(ctx context.Context) (int, error)
	GetConfigWithResourceVersion(ctx context.Context) (remote.Deployment, error)
}

// NewRemote returns new RemoteConfigReloader.
func NewRemote(log logrus.FieldLogger, statusReporter status.Reporter, deployCli DeploymentClient, restarter *Restarter, cfg config.Config, cfgVer int, resVerHolders ...ResourceVersionHolder) *RemoteConfigReloader {
	return &RemoteConfigReloader{
		log:            log,
		currentCfg:     cfg,
		resVersion:     cfgVer,
		interval:       cfg.ConfigWatcher.Remote.PollInterval,
		deployCli:      deployCli,
		resVerHolders:  resVerHolders,
		restarter:      restarter,
		statusReporter: statusReporter,
	}
}

// RemoteConfigReloader is responsible for reloading configuration from remote source.
type RemoteConfigReloader struct {
	log           logrus.FieldLogger
	interval      time.Duration
	resVerHolders []ResourceVersionHolder

	currentCfg config.Config
	resVersion int

	deployCli      DeploymentClient
	restarter      *Restarter
	statusReporter status.Reporter
}

// Do starts the remote config reloader.
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

			resVer, err := u.queryResourceVersion(ctx)
			if err != nil {
				u.log.Error(err.Error())
				continue
			}

			shouldUpdateCfg, err := u.compareResVer(resVer)
			if err != nil {
				u.log.Error(err.Error())
				continue
			}

			if !shouldUpdateCfg {
				u.log.Debugf("Config version (%d) is the same as the latest one. Skipping...", resVer)
				continue
			}

			u.log.Debugf("Config version changed (%d). Querying the latest config...", resVer)
			cfgBytes, resVer, err := u.queryConfig(ctx)
			if err != nil {
				wrappedErr := fmt.Errorf("while getting latest config: %w", err)
				u.log.Error(wrappedErr.Error())
				continue
			}

			cfgDiff, err := u.processNewConfig(ctx, cfgBytes, resVer)
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

func (u *RemoteConfigReloader) queryResourceVersion(ctx context.Context) (int, error) {
	resVer, err := u.deployCli.GetResourceVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("while getting resource version: %w", err)
	}

	return resVer, nil
}

func (u *RemoteConfigReloader) queryConfig(ctx context.Context) ([]byte, int, error) {
	deploy, err := u.deployCli.GetConfigWithResourceVersion(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("while getting deployment: %w", err)
	}

	var latestCfg config.Config
	err = yaml.Unmarshal([]byte(deploy.YAMLConfig), &latestCfg)
	if err != nil {
		return nil, 0, fmt.Errorf("while unmarshaling config: %w", err)
	}

	return []byte(deploy.YAMLConfig), deploy.ResourceVersion, nil
}

func (u *RemoteConfigReloader) compareResVer(newResVer int) (bool, error) {
	if newResVer == u.resVersion {
		return false, nil
	}
	if newResVer < u.resVersion {
		return false, fmt.Errorf("while comparing config versions: current config version (%d) is newer than the latest one (%d)", u.resVersion, newResVer)
	}

	return true, nil
}

type configDiff struct {
	shouldRestart bool
}

func (u *RemoteConfigReloader) processNewConfig(ctx context.Context, newCfgBytes []byte, newResVer int) (configDiff, error) {
	// another resource version check, because it can change between the first and second query
	shouldUpdate, err := u.compareResVer(newResVer)
	if err != nil {
		return configDiff{}, err
	}
	if !shouldUpdate {
		// this shouldn't happen, but better to be safe than sorry
		u.log.Debugf("After second query, config version (%d) is the same as the latest one. This shouldn't happen. Skipping...", newResVer)
		return configDiff{}, nil
	}
	u.setResourceVersionForAll(newResVer)

	newCfg, _, err := config.LoadWithDefaults([][]byte{newCfgBytes})
	if err != nil {
		return configDiff{}, fmt.Errorf("while loading new config: %w", err)
	}
	if newCfg == nil {
		return configDiff{}, fmt.Errorf("new config is nil")
	}

	changelog, err := diff.Diff(u.currentCfg, *newCfg, diff.DisableStructValues(), diff.SliceOrdering(false), diff.AllowTypeMismatch(true))
	if err != nil {
		return configDiff{}, fmt.Errorf("while diffing configs: %w", err)
	}

	if len(changelog) == 0 {
		u.log.Debugf("Config with higher version (%d) is the same as the latest one. No need to reload config", newResVer)
		if err := u.statusReporter.AckNewResourceVersion(ctx); err != nil {
			return configDiff{}, fmt.Errorf("while reporting config reload: %w", err)
		}
		return configDiff{}, nil
	}

	var paths []string
	for _, change := range changelog {
		paths = append(paths, fmt.Sprintf(`- "%s"`, strings.Join(change.Path, ".")))
	}
	u.log.Debugf("detected config changes on paths:\n%s", strings.Join(paths, "\n"))

	// TODO(https://github.com/kubeshop/botkube/issues/1012): check if notifications are enabled and if so, do not restart the app

	u.currentCfg = *newCfg
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

func (u *RemoteConfigReloader) SetResourceVersion(resourceVersion int) {
	u.setResourceVersionForAll(resourceVersion)
}
