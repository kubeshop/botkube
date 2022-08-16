package recommendation

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// AggregatedRunner contains multiple recommendations to run.
type AggregatedRunner struct {
	log             logrus.FieldLogger
	recommendations []Recommendation
}

func newAggregatedRunner(log logrus.FieldLogger, recommendations []Recommendation) AggregatedRunner {
	return AggregatedRunner{log: log, recommendations: recommendations}
}

// Do runs all recommendations within the set.
func (s AggregatedRunner) Do(ctx context.Context, event *events.Event) error {
	if len(s.recommendations) == 0 {
		s.log.Debug("No recommendations to run. Finishing...")
		return nil
	}

	if event == nil {
		return errors.New("event is nil")
	}

	errs := multierror.New()
	for _, r := range s.recommendations {
		s.log.Debugf("Running recommendation %q...", r.Name())
		result, err := r.Do(ctx, *event)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while running recommendation %q: %w", r.Name(), err))
		}

		event.Recommendations = append(event.Recommendations, result.Info...)
		event.Warnings = append(event.Warnings, result.Warnings...)
	}

	return errs.ErrorOrNil()
}
