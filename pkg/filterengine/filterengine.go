package filterengine

import (
	"fmt"
	"reflect"

	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
)

var (
	// DefaultFilterEngine contains default implementation for FilterEngine
	DefaultFilterEngine FilterEngine
)

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	Run(interface{}, events.Event) events.Event
	Register(Filter)
	ShowFilters() map[string]bool
	SetFilter(string, bool) error
}

type defaultFilters struct {
	FiltersMap map[Filter]bool
}

// Filter has method to run filter
type Filter interface {
	Run(interface{}, *events.Event)
}

func init() {
	DefaultFilterEngine = NewDefaultFilter()
}

// NewDefaultFilter creates new DefaultFilter object
func NewDefaultFilter() FilterEngine {
	var df defaultFilters
	df.FiltersMap = make(map[Filter]bool)
	return &df
}

// Run run the filters
func (f *defaultFilters) Run(object interface{}, event events.Event) events.Event {
	log.Logger.Debug("Filterengine running filters")
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
	log.Logger.Info("Registering the filter ", reflect.TypeOf(filter).Name())
	f.FiltersMap[filter] = true
}

// ShowFilters return map of filter name and status
func (f defaultFilters) ShowFilters() map[string]bool {
	fmap := make(map[string]bool)

	// Find filter struct name and set map
	for k, v := range f.FiltersMap {
		fmap[reflect.TypeOf(k).Name()] = v
	}
	return fmap
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
