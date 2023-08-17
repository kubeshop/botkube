package migrate

import (
	"time"

	"github.com/kubeshop/botkube/internal/cli/install/helm"
)

// Options holds migrate possible configuration options.
type Options struct {
	Timeout           time.Duration
	Token             string
	InstanceName      string `survey:"instanceName"`
	CloudDashboardURL string
	CloudAPIURL       string
	Namespace         string
	Label             string
	SkipConnect       bool
	SkipOpenBrowser   bool
	AutoApprove       bool
	ConfigExporter    ConfigExporterOptions
	HelmParams        helm.Config
	Watch             bool
}

// ConfigExporterOptions holds config exporter image configuration options.
type ConfigExporterOptions struct {
	Registry   string
	Repository string
	Tag        string

	Timeout    time.Duration
	PollPeriod time.Duration
}
