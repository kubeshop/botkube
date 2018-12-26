package filterengine

import (
	//"fmt"

	"github.com/infracloudio/kubeops/pkg/events"
	"github.com/infracloudio/kubeops/pkg/filterengine/filters"
	log "github.com/infracloudio/kubeops/pkg/logging"
)

var (
	DefaultFilterEngine FilterEngine

	// Create filters list
	// TODO: load this dynamically
	Filters = []Filter{
		filters.NewImageTagChecker(),
		filters.NewIngressValidator(),
	}
)

type FilterEngine interface {
	Run(interface{}, events.Event) events.Event
	Register(Filter)
}

type DefaultFilters struct {
	FiltersList []Filter
}

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
	return &DefaultFilters{}
}

// Run run the filters
func (f *DefaultFilters) Run(object interface{}, event events.Event) events.Event {
	log.Logger.Debug("Filterengine running filters")
	for _, f := range f.FiltersList {
		f.Run(object, &event)
	}
	return event
}

// Register filter to engine
func (f *DefaultFilters) Register(filter Filter) {
	log.Logger.Debug("Registering the filter", filter)
	f.FiltersList = append(f.FiltersList, filter)
}
