package sources

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/pkg/config"
)

type mergedEvents map[string]map[config.EventType]struct{}
type registrationHandler func(resource string) cache.SharedIndexInformer
type eventHandler func(ctx context.Context, resource string, sources []string, updateDiffs []string) func(obj interface{})

type route struct {
	source        string
	namespaces    config.Namespaces
	updateSetting config.UpdateSetting
}

func (r route) hasActionableUpdateSetting() bool {
	return len(r.updateSetting.Fields) > 0
}

type entry struct {
	event  config.EventType
	routes []route
}

// Router routes handled event types from registered
// informers to configured sources
type Router struct {
	log           logrus.FieldLogger
	mapper        meta.RESTMapper
	dynamicCli    dynamic.Interface
	table         map[string][]entry
	bindings      map[string]struct{}
	registrations map[string]registration
}

// NewRouter creates a new router to use for routing event types to registered informers.
func NewRouter(mapper meta.RESTMapper, dynamicCli dynamic.Interface, log logrus.FieldLogger) *Router {
	return &Router{
		log:           log,
		mapper:        mapper,
		dynamicCli:    dynamicCli,
		table:         make(map[string][]entry),
		bindings:      make(map[string]struct{}),
		registrations: make(map[string]registration),
	}
}

// AddAnyBindingsByName adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddAnyBindingsByName(c config.IdentifiableMap[config.ChannelBindingsByName]) {
	for _, byName := range c {
		r.AddAnyBindings(byName.Bindings)
	}
}

// AddAnyBindingsByID adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddAnyBindingsByID(c config.IdentifiableMap[config.ChannelBindingsByID]) {
	for _, byID := range c {
		r.AddAnyBindings(byID.Bindings)
	}
}

// AddAnyBindings adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddAnyBindings(b config.BotBindings) {
	for _, source := range b.Sources {
		r.bindings[source] = struct{}{}
	}
}

// AddAnySinkBindings adds source bindings names
// to dictate which source bindings the router should use.
func (r *Router) AddAnySinkBindings(b config.SinkBindings) {
	for _, source := range b.Sources {
		r.bindings[source] = struct{}{}
	}
}

// GetBoundSources returns the Sources the router uses
// based on preconfigured source binding names.
func (r *Router) GetBoundSources(sources config.IndexableMap[config.Sources]) config.IndexableMap[config.Sources] {
	out := make(config.IndexableMap[config.Sources])
	for name, t := range sources {
		if _, ok := r.bindings[name]; ok {
			out[name] = t
		}
	}
	return out
}

// BuildTable builds the routers routing table marking it ready
// to register, map and handle informer events.
func (r *Router) BuildTable(cfg *config.Config) *Router {
	sources := r.GetBoundSources(cfg.Sources)
	mergedEvents := mergeResourceEvents(sources)

	for resource, resourceEvents := range mergedEvents {
		eventRoutes := mergeEventRoutes(resource, sources)
		r.buildTable(resource, resourceEvents, eventRoutes)
	}
	r.log.Debugf("sources routing table: %+v", r.table)
	return r
}

// RegisterInformers register informers for all the resources that match the target events.
func (r *Router) RegisterInformers(targetEvents []config.EventType, handler registrationHandler) {
	resources := r.resourcesForEvents(targetEvents)
	for _, resource := range resources {
		r.registrations[resource] = registration{
			informer:   handler(resource),
			events:     r.resourceEvents(resource),
			log:        r.log,
			mapper:     r.mapper,
			dynamicCli: r.dynamicCli,
		}
	}
}

// MapWithEventsInformer allows resources to report on an event (srcEvent)
// that can only be observed by watching the general k8s v1/events resource
// using a different event (dstEvent).
//
// For example, you can report the "error" EventType for a resource
// by having the router watch/interrogate the "warning" EventType
// reported by the v1/events resource.
func (r *Router) MapWithEventsInformer(srcEvent config.EventType, dstEvent config.EventType, handler registrationHandler) {
	srcResources := r.resourcesForEvents([]config.EventType{srcEvent})
	if len(srcResources) == 0 {
		return
	}

	dstResource := "v1/events"
	r.registrations[dstResource] = registration{
		informer:        handler(dstResource),
		events:          []config.EventType{dstEvent},
		mappedResources: srcResources,
		mappedEvent:     srcEvent,
		log:             r.log,
		mapper:          r.mapper,
		dynamicCli:      r.dynamicCli}
}

// HandleEvent allows router clients to create handlers that are
// triggered for a target event.
func (r *Router) HandleEvent(ctx context.Context, target config.EventType, handlerFn eventHandler) {
	for resource, informer := range r.registrations {
		if informer.canHandleEvent(target.String()) {
			sourceRoutes := r.sourceRoutes(resource, target)
			informer.handleEvent(ctx, resource, target, sourceRoutes, handlerFn)
		}
	}
}

// HandleMappedEvent allows router clients to create handlers that are
// triggered for a target mapped event.
func (r *Router) HandleMappedEvent(ctx context.Context, targetEvent config.EventType, handlerFn eventHandler) {
	if informer, ok := r.mappedInformer(targetEvent); ok {
		informer.handleMapped(ctx, targetEvent, r.table, handlerFn)
	}
}

func mergeResourceEvents(sources config.IndexableMap[config.Sources]) mergedEvents {
	out := map[string]map[config.EventType]struct{}{}
	for _, srcGroupCfg := range sources {
		for _, resource := range srcGroupCfg.Kubernetes.Resources {
			if _, ok := out[resource.Name]; !ok {
				out[resource.Name] = make(map[config.EventType]struct{})
			}
			for _, e := range flattenEvents(resource.Events) {
				out[resource.Name][e] = struct{}{}
			}
		}
	}
	return out
}

func mergeEventRoutes(resource string, sources config.IndexableMap[config.Sources]) map[config.EventType][]route {
	out := make(map[config.EventType][]route)
	for srcGroupName, srcGroupCfg := range sources {
		for _, r := range srcGroupCfg.Kubernetes.Resources {
			for _, e := range flattenEvents(r.Events) {
				switch {
				case resource == r.Name && e == config.UpdateEvent:
					out[e] = append(out[e], route{
						source:     srcGroupName,
						namespaces: r.Namespaces,
						updateSetting: config.UpdateSetting{
							Fields:      r.UpdateSetting.Fields,
							IncludeDiff: r.UpdateSetting.IncludeDiff,
						}})
				case resource == r.Name && e != config.UpdateEvent:
					out[e] = append(out[e], route{source: srcGroupName, namespaces: r.Namespaces})
				}
			}
		}
	}
	return out
}

func (r *Router) buildTable(resource string, events map[config.EventType]struct{}, routes map[config.EventType][]route) {
	for evt := range events {
		if _, ok := r.table[resource]; !ok {
			r.table[resource] = []entry{{
				event:  evt,
				routes: routes[evt],
			}}
		} else {
			r.table[resource] = append(r.table[resource], entry{event: evt, routes: routes[evt]})
		}
	}
}

func (r *Router) sourceRoutes(resource string, targetEvent config.EventType) []route {
	return sourceRoutes(r.table, resource, targetEvent)
}

func sourceRoutes(routeTable map[string][]entry, targetResource string, targetEvent config.EventType) []route {
	var out []route
	for _, routedEvent := range routeTable[targetResource] {
		if routedEvent.event == targetEvent {
			out = append(out, routedEvent.routes...)
		}
	}
	return out
}

func (r *Router) resourceEvents(resource string) []config.EventType {
	var out []config.EventType
	for _, routedEvent := range r.table[resource] {
		out = append(out, routedEvent.event)
	}
	return out
}

func (r *Router) resourcesForEvents(targets []config.EventType) []string {
	var out []string
	for _, target := range targets {
		for resource, routedEvents := range r.table {
			for _, routedEvent := range routedEvents {
				if routedEvent.event == target {
					out = append(out, resource)
					break
				}
			}
		}
	}
	return out
}

func (r *Router) mappedInformer(event config.EventType) (registration, bool) {
	for _, informer := range r.registrations {
		if informer.mappedEvent == event {
			return informer, true
		}
	}
	return registration{}, false
}

func flattenEvents(events []config.EventType) []config.EventType {
	var out []config.EventType
	for _, event := range events {
		if event == config.AllEvent {
			out = append(out, []config.EventType{config.CreateEvent, config.UpdateEvent, config.DeleteEvent, config.ErrorEvent}...)
		} else {
			out = append(out, event)
		}
	}
	return out
}
