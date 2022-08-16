package filterengine

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/events"
)

// DefaultFilterEngine is a default implementation of the Filter Engine
type DefaultFilterEngine struct {
	log logrus.FieldLogger

	filters map[string]RegisteredFilter
}

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	Run(context.Context, events.Event) events.Event
	Register(...Filter)
	RegisteredFilters() []RegisteredFilter
	SetFilter(string, bool) error
}

// RegisteredFilter contains details about registered filter
type RegisteredFilter struct {
	Enabled bool
	Filter
}

// Filter has method to run filter
type Filter interface {
	Run(context.Context, *events.Event) error
	Name() string
	Describe() string
}

// New creates new DefaultFilterEngine instance.
func New(log logrus.FieldLogger) *DefaultFilterEngine {
	return &DefaultFilterEngine{
		log:     log,
		filters: make(map[string]RegisteredFilter),
	}
}

// Run runs the registered filters always iterating over a slice of filters with sorted keys
func (f *DefaultFilterEngine) Run(ctx context.Context, event events.Event) events.Event {
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

// Register filter(s) to engine
func (f *DefaultFilterEngine) Register(filters ...Filter) {
	for _, filter := range filters {
		f.log.Infof("Registering filter %q", filter.Name())
		f.filters[filter.Name()] = RegisteredFilter{
			Filter:  filter,
			Enabled: true,
		}
	}
}

// RegisteredFilters returns sorted slice of registered filters
func (f DefaultFilterEngine) RegisteredFilters() []RegisteredFilter {
	var keys []string
	for key := range f.filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var registeredFilters []RegisteredFilter
	for _, key := range keys {
		registeredFilters = append(registeredFilters, f.filters[key])
	}

	return registeredFilters
}

// SetFilter sets filter value in FilterMap to enable or disable filter
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
