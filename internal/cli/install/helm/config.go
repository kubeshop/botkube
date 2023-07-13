package helm

import (
	"helm.sh/helm/v3/pkg/cli/values"
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
