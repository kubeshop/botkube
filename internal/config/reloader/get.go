package reloader

import (
	"github.com/sirupsen/logrus"
	"time"
)

// Get returns Reloader based on remoteCfgEnabled flag.
func Get(remoteCfgEnabled bool, log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) Reloader {
	if remoteCfgEnabled {
		return NewRemote(log, interval, deployCli, resVerHolders...)
	}

	return NewNoopReloader()
}
