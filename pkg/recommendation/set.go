package recommendation

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// Set contains multiple recommendations to run.
type Set struct {
	log logrus.FieldLogger
	set map[string]Recommendation
}

func newRecommendationsSet(log logrus.FieldLogger, set map[string]Recommendation) Set {
	return Set{log: log, set: set}
}

// Run runs all recommendations within the set.
func (s Set) Run(ctx context.Context, event *events.Event) error {
	if len(s.set) == 0 {
		s.log.Debug("No recommendations to run. Finishing...")
		return nil
	}

	if event == nil {
		return errors.New("event is nil")
	}

	errs := multierror.New()
	for _, key := range s.sortedKeys() {
		recommendation := s.set[key]
		s.log.Debugf("Running recommendation %q...", recommendation.Name())
		result, err := recommendation.Do(ctx, *event)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while running recommendation %q: %w", recommendation.Name(), err))
		}

		event.Recommendations = append(event.Recommendations, result.Info...)
		event.Warnings = append(event.Warnings, result.Warnings...)
	}

	return errs.ErrorOrNil()
}

func (s Set) sortedKeys() []string {
	var keys []string
	for key := range s.set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
