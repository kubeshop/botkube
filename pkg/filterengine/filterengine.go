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

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/events"
)

// DefaultFilterEngine is a default implementation of the Filter Engine
type DefaultFilterEngine struct {
	log logrus.FieldLogger

	// TODO: Change map key to filter name to be able to SetFilter without iterating through the whole map
	FiltersMap map[Filter]bool
}

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	Run(context.Context, interface{}, events.Event) events.Event
	Register(...Filter)
	ShowFilters() map[Filter]bool
	SetFilter(string, bool) error
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
		log:        log,
		FiltersMap: make(map[Filter]bool),
	}
}

// Run runs the registered filters
func (f *DefaultFilterEngine) Run(ctx context.Context, object interface{}, event events.Event) events.Event {
	f.log.Debug("Running registered filters")
	for filter, enabled := range f.FiltersMap {
		if !enabled {
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
		f.FiltersMap[filter] = true
	}
}

// ShowFilters return map of filter name and status
// TODO: This method is used for printing filters, but the order of the list is not preserved. Fix it.
func (f DefaultFilterEngine) ShowFilters() map[Filter]bool {
	return f.FiltersMap
}

// SetFilter sets filter value in FilterMap to enable or disable filter
func (f *DefaultFilterEngine) SetFilter(name string, flag bool) error {
	// Find filter struct name
	for k := range f.FiltersMap {
		if k.Name() == name {
			f.FiltersMap[k] = flag
			return nil
		}
	}
	return fmt.Errorf("couldn't find filter with name %q", name)
}
