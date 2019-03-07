package filterengine

import (
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine/filters"
	log "github.com/infracloudio/botkube/pkg/logging"
)

var (
	// DefaultFilterEngine contains default implementation for FilterEngine
	DefaultFilterEngine FilterEngine

	// Filters contains the lists of available filters
	// TODO: load this dynamically
	Filters = []Filter{
		filters.NewImageTagChecker(),
		filters.NewIngressValidator(),
		filters.NewPodLabelChecker(),
	}
)

// FilterEngine has methods to register and run filters
type FilterEngine interface {
	Run(interface{}, events.Event) events.Event
	Register(Filter)
}

type defaultFilters struct {
	FiltersList []Filter
}

// Filter has method to run filter
type Filter interface {
	Run(interface{}, *events.Event)
}

func init() {
	DefaultFilterEngine = NewDefaultFilter()

	// Register filters
	for _, f := range Filters {
		DefaultFilterEngine.Register(f)
	}
}

// NewDefaultFilter creates new DefaultFilter object
func NewDefaultFilter() FilterEngine {
	return &defaultFilters{}
}

// Run run the filters
func (f *defaultFilters) Run(object interface{}, event events.Event) events.Event {
	log.Logger.Debug("Filterengine running filters")
	for _, f := range f.FiltersList {
		f.Run(object, &event)
	}
	return event
}

// Register filter to engine
func (f *defaultFilters) Register(filter Filter) {
	log.Logger.Debug("Registering the filter", filter)
	f.FiltersList = append(f.FiltersList, filter)
}
