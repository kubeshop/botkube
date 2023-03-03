package config

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
)

type ConfigUpdater interface {
	Do(ctx context.Context) error
}

type ResourceVersionHolder interface {
	SetResourceVersion(int)
}

func GetConfigUpdater(remoteCfgEnabled bool, log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) ConfigUpdater {
	if remoteCfgEnabled {
		return NewConfigUpdater(log, interval, deployCli, resVerHolders...)
	}

	return &noopConfigUpdater{}
}

func NewConfigUpdater(log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) ConfigUpdater {
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
			}

			u.latestCfg = cfg
			u.setResourceVersionForAll(resVer)
			u.log.Debugf("Successfully set config version %d.", resVer)
		}
	}
}

func (u *GraphQLConfigUpdater) queryConfig(ctx context.Context) (config.Config, int, error) {
	deploy, err := u.deployCli.GetDeployment(ctx)
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
