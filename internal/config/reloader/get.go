package reloader

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/status"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	typeKey = "type"
)

// Get returns Reloader based on remoteCfgEnabled flag.
func Get(remoteCfgEnabled bool, log logrus.FieldLogger, statusReporter status.Reporter, deployCli DeploymentClient, dynamicCli dynamic.Interface, restarter *Restarter, reporter analytics.Reporter, cfg config.Config, cfgVer int, resVerHolders ...ResourceVersionHolder) (Reloader, error) {
	if remoteCfgEnabled {
		log = log.WithField(typeKey, "remote")
		return NewRemote(log, statusReporter, deployCli, restarter, cfg, cfgVer, resVerHolders...), nil
	}

	log = log.WithField(typeKey, "in-cluster")
	return NewInClusterConfigReloader(log, dynamicCli, cfg.ConfigWatcher, restarter, reporter)
}
