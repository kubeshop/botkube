package analytics

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
func NewReporter() Reporter {
	if APIKey == "" {
		return &NoopReporter{}
	}
	r, err := NewSegmentReporter(APIKey)
	if err != nil {
		return &NoopReporter{}
	}
	return r
}
