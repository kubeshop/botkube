package config

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
)

// ConfigUpdater is an interface for updating configuration.
type ConfigUpdater interface {
	Do(ctx context.Context) error
}

// ResourceVersionHolder is an interface for holding resource version with ability to set it.
type ResourceVersionHolder interface {
	SetResourceVersion(int)
}

// GetConfigUpdater returns ConfigUpdater based on remoteCfgEnabled flag.
func GetConfigUpdater(remoteCfgEnabled bool, log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) ConfigUpdater {
	if remoteCfgEnabled {
		return newRemoteConfigUpdater(log, interval, deployCli, resVerHolders...)
	}

	return &noopConfigUpdater{}
}

// newRemoteConfigUpdater returns new ConfigUpdater.
func newRemoteConfigUpdater(log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) ConfigUpdater {
	return &GraphQLConfigUpdater{
		log:           log,
		interval:      interval,
		deployCli:     deployCli,
		resVerHolders: resVerHolders,
	}
}

type GraphQLConfigUpdater struct {
	log           logrus.FieldLogger
	interval      time.Duration
	resVerHolders []ResourceVersionHolder

	latestCfg  config.Config
	resVersion int

	deployCli DeploymentClient
}

func (u *GraphQLConfigUpdater) Do(ctx context.Context) error {
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

			u.latestCfg = cfg
			u.setResourceVersionForAll(resVer)
			u.log.Debugf("Successfully set newer config version (%d)", resVer)
		}
	}
}

func (u *GraphQLConfigUpdater) queryConfig(ctx context.Context) (config.Config, int, error) {
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

func (u *GraphQLConfigUpdater) setResourceVersionForAll(resVersion int) {
	u.resVersion = resVersion
	for _, h := range u.resVerHolders {
		h.SetResourceVersion(u.resVersion)
	}
}

type noopConfigUpdater struct{}

func (u *noopConfigUpdater) Do(ctx context.Context) error {
	return nil
}
