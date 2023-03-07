package reloader

import (
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

// Get returns Reloader based on remoteCfgEnabled flag.
func Get(remoteCfgEnabled bool, log logrus.FieldLogger, deployCli DeploymentClient, restarter *Restarter, cfg config.Config, cfgVer int, resVerHolders ...ResourceVersionHolder) Reloader {
	if remoteCfgEnabled {
		return NewRemote(log, deployCli, restarter, cfg, cfgVer, resVerHolders...)
	}

	return NewNoopReloader()
}
