package batched

import "sync"

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

	key := in.PluginName
	sourceProps := d.heartbeatProperties.Sources[key]
	sourceProps.Events = append(sourceProps.Events, in)
	sourceProps.EventsCount = len(sourceProps.Events)

	d.heartbeatProperties.Sources[key] = sourceProps
	d.heartbeatProperties.EventsCount++
}
