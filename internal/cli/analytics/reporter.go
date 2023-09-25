package analytics

import "github.com/kubeshop/botkube/internal/cli"

const (
	defaultCliVersion = "v9.99.9-dev"
)

var (
	// APIKey contains the API key for external analytics service. It is set during application build.
	APIKey string = ""
)

// Reporter defines behavior for reporting analytics.
type Reporter interface {
	ReportCommand(cmd string) error
	ReportError(err error, cmd string) error
	Close()
}

// NewReporter creates a new Reporter instance.
func GetReporter() Reporter {
	if APIKey == "" {
		return &NoopReporter{}
	}

	conf := cli.NewConfig()
	if conf.IsTelemetryDisabled() {
		return &NoopReporter{}
	}

	// Create segment reporter if telemetry enabled and API key is set
	r, err := NewSegmentReporter(APIKey)
	if err != nil {
		// do not crash on telemetry errors
		return &NoopReporter{}
	}
	return r
}
