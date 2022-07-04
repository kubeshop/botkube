package analytics

var _ Reporter = &NoopReporter{}

type NoopReporter struct{}

func NewNoopReporter() *NoopReporter {
	return &NoopReporter{}
}
