package filterengine

import (
	"context"
	"fmt"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/maputil"
)

// DefaultFilterEngine is a default implementation of the Filter Engine.
type DefaultFilterEngine struct {
	log logrus.FieldLogger

	filters map[string]RegisteredFilter
}

// FilterEngine has methods to register and run filters.
type FilterEngine interface {
	Run(context.Context, event.Event) event.Event
	Register(...RegisteredFilter)
	RegisteredFilters() []RegisteredFilter
	SetFilter(string, bool) error
}

// RegisteredFilter contains details about registered filter.
type RegisteredFilter struct {
	Enabled bool
	Filter
}

// Filter defines an event filter.
type Filter interface {
	Run(context.Context, *event.Event) error
	Name() string
	Describe() string
}

// New creates new DefaultFilterEngine instance..
func New(log logrus.FieldLogger) *DefaultFilterEngine {
	return &DefaultFilterEngine{
		log:     log,
		filters: make(map[string]RegisteredFilter),
	}
}

// Run runs the registered filters always iterating over a slice of filters with sorted keys.
func (f *DefaultFilterEngine) Run(ctx context.Context, event event.Event) event.Event {
	f.log.Debug("Running registered filters")
	filters := f.RegisteredFilters()
	f.log.Debugf("registered filters: %+v", filters)

	for _, filter := range filters {
		if !filter.Enabled {
			continue
		}

		err := filter.Run(ctx, &event)
		if err != nil {
			f.log.Errorf("while running filter %q: %w", filter.Name(), err)
		}
		f.log.Debugf("ran filter name: %q, event was skipped: %t", filter.Name(), event.Skip)
	}
	return event
}

// Register filter(s) to engine.
func (f *DefaultFilterEngine) Register(filters ...RegisteredFilter) {
	for _, filter := range filters {
		f.log.Infof("Registering filter %q (enabled: %t)...", filter.Name(), filter.Enabled)
		f.filters[filter.Name()] = filter
	}
}

// RegisteredFilters returns sorted slice of registered filters.
func (f *DefaultFilterEngine) RegisteredFilters() []RegisteredFilter {
	var registeredFilters []RegisteredFilter
	for _, key := range maputil.SortKeys(f.filters) {
		registeredFilters = append(registeredFilters, f.filters[key])
	}

	return registeredFilters
}

// SetFilter sets filter value in FilterMap to enable or disable filter.
func (f *DefaultFilterEngine) SetFilter(name string, flag bool) error {
	// Find filter struct name
	filter, ok := f.filters[name]
	if !ok {
		return fmt.Errorf("couldn't find filter with name %q", name)
	}

	filter.Enabled = flag
	f.filters[name] = filter
	return nil
}
