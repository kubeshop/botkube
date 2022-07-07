package analytics

var _ Reporter = &NoopReporter{}

type NoopReporter struct{}

func NewNoopReporter() *NoopReporter {
	return &NoopReporter{}
}

func (n NoopReporter) RegisterIdentity(_ Identity) error {
	return nil
}
