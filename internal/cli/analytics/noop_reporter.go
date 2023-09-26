package analytics

var _ Reporter = &NoopReporter{}

// NoopReporter is a no-op implementation of the Reporter interface.
type NoopReporter struct{}

// ReportCommand reports a command to the analytics service.
func (r *NoopReporter) ReportCommand(cmd string) error {
	return nil
}

// ReportError reports an error to the analytics service.
func (r *NoopReporter) ReportError(err error, cmd string) error {
	return nil
}

// Close closes the reporter.
func (r *NoopReporter) Close() {}
