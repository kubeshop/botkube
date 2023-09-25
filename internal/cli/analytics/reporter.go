package analytics

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
)

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
func GetReporter(cmd cobra.Command) Reporter {
	if APIKey == "" {
		printWhenVerboseEnabled(cmd, "Telemetry disabled - no API key")
		return &NoopReporter{}
	}

	conf := cli.NewConfig()
	if conf.IsTelemetryDisabled() {
		printWhenVerboseEnabled(cmd, "Telemetry disabled - config")
		return &NoopReporter{}
	}

	// Create segment reporter if telemetry enabled and API key is set
	r, err := NewSegmentReporter(APIKey)
	if err != nil {
		// do not crash on telemetry errors
		printWhenVerboseEnabled(cmd, "Telemetry disabled - API key wasn't accepted")
		return &NoopReporter{}
	}

	printWhenVerboseEnabled(cmd, "Telemetry enabled")
	return r
}

func printWhenVerboseEnabled(cmd cobra.Command, s string) {
	if cli.VerboseMode.IsEnabled() {
		cmd.Println(s)
	}
}
