// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package filterengine

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/events"
)

// DefaultFilterEngine is a default implementation of the Filter Engine
type DefaultFilterEngine struct {
	log logrus.FieldLogger

	filters map[string]RegisteredFilter
}

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	// TODO: Why `Run` method takes object as input argument, if event already contains it as well? Refactor it if possible
	Run(context.Context, interface{}, events.Event) events.Event
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
	Run(context.Context, interface{}, *events.Event) error
	Name() string
	Describe() string
}

// New creates new DefaultFilterEngine object
func New(log logrus.FieldLogger) *DefaultFilterEngine {
	return &DefaultFilterEngine{
		log:     log,
		filters: make(map[string]RegisteredFilter),
	}
}

// Run runs the registered filters always iterating over a slice of filters with sorted keys
func (f *DefaultFilterEngine) Run(ctx context.Context, object interface{}, event events.Event) events.Event {
	f.log.Debug("Running registered filters")
	filters := f.RegisteredFilters()
	for _, filter := range filters {
		if !filter.Enabled {
			continue
		}

		err := filter.Run(ctx, object, &event)
		if err != nil {
			f.log.Errorf("while running filter %q: %w", filter.Name(), err)
		}
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
