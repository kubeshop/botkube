package batched

import (
	"sync"
)

const (

	// Segment limits the API calls to 32kB per request: https://segment.com/docs/connections/sources/catalog/libraries/server/http-api/
	// We save 2kB (2048 characters) for general metadata. The rest of 30kB we can spend for sending source event details.
	// Average event details size is 300 characters. So in theory we could include 30*1024/300=102.4 events.
	// As the plugin name and additional labels don't have fixed size, we limit the number of events to 75 to be on the safe side.
	maxEventDetailsCount = 75
)

// Data is a struct that holds data for batched reporting
type Data struct {
	mutex sync.RWMutex

	defaultTimeWindowInHours int
	heartbeatProperties      HeartbeatProperties
}

func NewData(defaultTimeWindowInHours int) *Data {
	return &Data{
		defaultTimeWindowInHours: defaultTimeWindowInHours,
		heartbeatProperties: HeartbeatProperties{
			TimeWindowInHours: defaultTimeWindowInHours,
			EventsCount:       0,
			Sources:           make(map[string]SourceProperties),
		}}
}

func (d *Data) IncrementTimeWindowInHours() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.heartbeatProperties.TimeWindowInHours++
}

func (d *Data) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.heartbeatProperties.TimeWindowInHours = d.defaultTimeWindowInHours
	d.heartbeatProperties.Sources = make(map[string]SourceProperties)
	d.heartbeatProperties.EventsCount = 0
}

func (d *Data) HeartbeatProperties() HeartbeatProperties {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.heartbeatProperties
}

func (d *Data) AddSourceEvent(in SourceEvent) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.heartbeatProperties.EventsCount++

	key := in.PluginName
	sourceProps := d.heartbeatProperties.Sources[key]
	sourceProps.EventsCount++
	if d.heartbeatProperties.EventsCount <= maxEventDetailsCount {
		// save event details only if we didn't exceed the limit
		sourceProps.Events = append(sourceProps.Events, in)
	}
	d.heartbeatProperties.Sources[key] = sourceProps
}
