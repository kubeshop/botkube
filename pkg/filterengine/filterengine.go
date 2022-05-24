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
	"fmt"
	"reflect"

	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
)

var (
	// DefaultFilterEngine contains default implementation for FilterEngine
	DefaultFilterEngine FilterEngine
)

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	Run(interface{}, events.Event) events.Event
	Register(Filter)
	RegisterMany([]Filter)
	ShowFilters() map[Filter]bool
	SetFilter(string, bool) error
}

type defaultFilters struct {
	FiltersMap map[Filter]bool
}

// Filter has method to run filter
type Filter interface {
	Run(interface{}, *events.Event)
	Describe() string
}

// NewDefaultFilter creates new DefaultFilter object
func NewDefaultFilter() FilterEngine {
	var df defaultFilters
	df.FiltersMap = make(map[Filter]bool)
	return &df
}

// Run run the filters
func (f *defaultFilters) Run(object interface{}, event events.Event) events.Event {
	log.Debug("Filterengine running filters")
	// Run registered filters
	for k, v := range f.FiltersMap {
		if v {
			k.Run(object, &event)
		}
	}
	return event
}

// Register filter to engine
func (f *defaultFilters) Register(filter Filter) {
	log.Info("Registering the filter ", reflect.TypeOf(filter).Name())
	f.FiltersMap[filter] = true
}

// RegisterMany registers multiple filters
func (f *defaultFilters) RegisterMany(filters []Filter) {
	for _, filter := range filters {
		f.Register(filter)
	}
}

// ShowFilters return map of filter name and status
func (f defaultFilters) ShowFilters() map[Filter]bool {
	return f.FiltersMap
}

// SetFilter sets filter value in FilterMap to enable or disable filter
func (f *defaultFilters) SetFilter(name string, flag bool) error {
	// Find filter struct name
	for k := range f.FiltersMap {
		if reflect.TypeOf(k).Name() == name {
			f.FiltersMap[k] = flag
			return nil
		}
	}
	return fmt.Errorf("Invalid filter name %s", name)
}
