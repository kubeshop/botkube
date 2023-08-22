package helm

import (
	"helm.sh/helm/v3/pkg/cli/values"
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

// Config holds Helm configuration parameters.
type Config struct {
	ReleaseName  string
	ChartName    string
	Version      string
	RepoLocation string
	AutoApprove  bool

	Namespace                string
	SkipCRDs                 bool
	DisableHooks             bool
	DryRun                   bool
	Force                    bool
	Atomic                   bool
	SubNotes                 bool
	Description              string
	DisableOpenAPIValidation bool
	DependencyUpdate         bool
	Values                   values.Options

	UpgradeConfig
}

// UpgradeConfig holds upgrade related settings.
type UpgradeConfig struct {
	ResetValues bool
	ReuseValues bool
}
