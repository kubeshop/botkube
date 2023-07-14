package install

import (
	"time"

	"github.com/kubeshop/botkube/internal/cli/install/helm"
)

const (
	// StableVersionTag tag used to select stable Helm chart repository.
	StableVersionTag = "@stable"
	// LocalVersionTag tag used to select local Helm chart repository.
	LocalVersionTag = "@local"
	// LatestVersionTag tag used to select the latest version from the Helm chart repository.
	LatestVersionTag = "@latest"
	// Namespace in which Botkube is installed.
	Namespace = "botkube"
	// ReleaseName defines Botkube Helm chart release name.
	ReleaseName = "botkube"
	// HelmRepoStable URL of the stable Botkube Helm charts repository.
	HelmRepoStable = "https://charts.botkube.io/"
	// HelmChartName represents Botkube Helm chart name in a given Helm repository.
	HelmChartName = "botkube"
	// LocalChartsPath path to Helm charts in botkube repository.
	LocalChartsPath = "./helm/"
)

// Config holds parameters for Botkube installation on cluster.
type Config struct {
	Kubeconfig string
	HelmParams helm.Config
	Watch      bool
	Timeout    time.Duration
}