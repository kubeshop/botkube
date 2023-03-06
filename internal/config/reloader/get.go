package reloader

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Get returns Reloader based on remoteCfgEnabled flag.
func Get(remoteCfgEnabled bool, log logrus.FieldLogger, interval time.Duration, deployCli DeploymentClient, resVerHolders ...ResourceVersionHolder) Reloader {
	if remoteCfgEnabled {
		return NewRemote(log, interval, deployCli, resVerHolders...)
	}

	return NewNoopReloader()
}
