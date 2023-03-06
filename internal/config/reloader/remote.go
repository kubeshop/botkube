package reloader

import (
	"context"
	"fmt"
	"time"

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
func NewRemote(log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) *RemoteConfigReloader {
	return &RemoteConfigReloader{
		log:           log,
		interval:      interval,
		deployCli:     deployCli,
		resVerHolders: resVerHolders,
	}
}

type RemoteConfigReloader struct {
	log           logrus.FieldLogger
	interval      time.Duration
	resVerHolders []ResourceVersionHolder

	latestCfg  config.Config
	resVersion int

	deployCli DeploymentClient
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

			if resVer == u.resVersion {
				u.log.Debugf("Config version (%d) is the same as the latest one. Skipping...", resVer)
				continue
			}

			// TODO: check diff


			u.latestCfg = cfg
			u.setResourceVersionForAll(resVer)
			u.log.Debugf("Successfully set newer config version (%d)", resVer)
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

func (u *RemoteConfigReloader) setResourceVersionForAll(resVersion int) {
	u.resVersion = resVersion
	for _, h := range u.resVerHolders {
		h.SetResourceVersion(u.resVersion)
	}
}
