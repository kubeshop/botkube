package helm

import (
	"helm.sh/helm/v3/pkg/cli/values"
)

const (
	repositoryCache = "/tmp/helm"
	helmDriver      = "secrets"
)

// Config holds Helm configuration parameters.
type Config struct {
	ReleaseName  string
	ChartName    string
	Version      string
	RepoLocation string

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
