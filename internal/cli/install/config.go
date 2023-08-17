package install

import (
	"time"

	"github.com/kubeshop/botkube/internal/cli/install/helm"
)

// Config holds parameters for Botkube installation on cluster.
type Config struct {
	Kubeconfig string
	HelmParams helm.Config
	Watch      bool
	Timeout    time.Duration
}
