package migrate

import "time"

// Options holds migrate possible configuration options.
type Options struct {
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
}

// ConfigExporterOptions holds config exporter image configuration options.
type ConfigExporterOptions struct {
	Registry   string
	Repository string
	Tag        string

	Timeout    time.Duration
	PollPeriod time.Duration
}
