package migrate

// Options holds migrate possible configuration options.
type Options struct {
	Token             string
	InstanceName      string `survey:"instanceName"`
	CloudDashboardURL string
	CloudAPIURL       string
	Namespace         string
	Label             string
	SkipConnect       bool
}
