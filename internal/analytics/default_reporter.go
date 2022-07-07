package analytics

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
)

var (
	// APIKey contains the API key for external analytics service. It is set during application build.
	APIKey string
)

var _ Reporter = &DefaultReporter{}

type DefaultReporter struct {
	log logrus.FieldLogger
	cli segment.Client

	identity *Identity
}

type CleanupFn func() error

func NewDefaultReporter(log logrus.FieldLogger) (*DefaultReporter, CleanupFn, error) {
	cli, err := segment.NewWithConfig(APIKey, segment.Config{
		Logger:  newLoggerAdapter(log),
		Verbose: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("while creating new Analytics Client: %w", err)
	}

	cleanupFn := func() error {
		log.Info("Closing...") // TODO: Should we use Debug?
		return cli.Close()
	}

	return &DefaultReporter{
			log: log,
			cli: cli,
		},
		cleanupFn,
		nil
}

func (r *DefaultReporter) RegisterIdentity(identity Identity) error {
	err := r.cli.Enqueue(segment.Identify{
		AnonymousId: identity.Installation.ID,
		Traits:      identity.Installation.TraitsMap(),
	})
	if err != nil {
		return fmt.Errorf("while enqueuing itentify message: %w", err)
	}

	err = r.cli.Enqueue(segment.Group{
		AnonymousId: identity.Installation.ID,
		GroupId:     identity.Cluster.ID,
		Traits:      identity.Cluster.TraitsMap(),
	})
	if err != nil {
		return fmt.Errorf("while enqueuing group message: %w", err)
	}

	r.identity = &identity
	return nil
}

type loggerAdapter struct {
	log logrus.FieldLogger
}

func newLoggerAdapter(log logrus.FieldLogger) *loggerAdapter {
	return &loggerAdapter{log: log}
}

func (l *loggerAdapter) Logf(format string, args ...interface{}) {
	l.log.Infof(format, args) // TODO: Should we use Debug?
}

func (l *loggerAdapter) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args)
}
