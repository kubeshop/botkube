package filterengine

import (
	"path/filepath"
	"plugin"

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
	Register(plugin.Symbol)
}

type defaultFilters struct {
	FiltersList []plugin.Symbol
}

// Filter has method to run filter
type Filter interface {
	Run(interface{}, *events.Event)
}

func init() {
	DefaultFilterEngine = NewDefaultFilter()

	// Find filter plugins
	// FIXME: add relative path
	plugins, _ := filepath.Glob("PATH/*.so")

	// Register filters
	for _, p := range plugins {
		log.Logger.Debugf("registering %s filter", p)
		plug, err := plugin.Open(p)
		if err != nil {
			log.Logger.Fatal(err)
		}

		filterRun, err := plug.Lookup("Filter")
		if err != nil {
			log.Logger.Fatal(err)
		}

		DefaultFilterEngine.Register(filterRun)
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
		var filter Filter
		filter, ok := f.(Filter)
		if !ok {
			log.Logger.Warn("Error while parsing plugin")
			continue
		}
		filter.Run(object, &event)
	}
	return event
}

// Register filter to engine
func (f *defaultFilters) Register(filter plugin.Symbol) {
	log.Logger.Debug("Registering the filter", filter)
	f.FiltersList = append(f.FiltersList, filter)
}
