package sources

import (
	"context"
	"fmt"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

type mergedEvents map[string]map[config.EventType]struct{}

type Router struct {
	log        logrus.FieldLogger
	mapper     meta.RESTMapper
	dynamicCli dynamic.Interface
	table      map[string][]RoutedEvent
	bindings   map[string]struct{}
	informers  map[string]Informer
}

type Routes struct {
	source     string
	namespaces []string
}

type RoutedEvent struct {
	event  config.EventType
	routes []Routes
}

func NewSourcesRouter(mapper meta.RESTMapper, dynamicCli dynamic.Interface, log logrus.FieldLogger) *Router {
	return &Router{
		log:        log,
		mapper:     mapper,
		dynamicCli: dynamicCli,
		table:      make(map[string][]RoutedEvent),
		bindings:   make(map[string]struct{}),
		informers:  make(map[string]Informer),
	}
}

func (r *Router) AddAnySlackBindings(c config.IdentifiableMap[config.ChannelBindingsByName]) {
	for _, name := range c {
		for _, source := range name.Bindings.Sources {
			r.bindings[source] = struct{}{}
		}
	}
}

func (r *Router) BuildTable(cfg *config.Config) *Router {
	sources := r.GetBoundSources(cfg.Sources)
	mergedEvents := mergeResourceEvents(sources)

	for resource, resourceEvents := range mergedEvents {
		eventRoutes := mergeEventRoutes(resource, sources)
		fmt.Printf("\n\nCOLLECTED_ROUTES: %+v\n\n", eventRoutes)
		r.buildTable(resource, resourceEvents, eventRoutes)
	}
	r.log.Debugf("sources routing table: %+v", r.table)
	return r
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

func mergeEventRoutes(resource string, sources config.IndexableMap[config.Sources]) map[config.EventType][]Routes {
	out := make(map[config.EventType][]Routes)
	for srcGroupName, srcGroupCfg := range sources {
		for _, r := range srcGroupCfg.Kubernetes.Resources {
			for _, e := range flattenEvents(r.Events) {
				if resource == r.Name {
					out[e] = append(out[e], Routes{source: srcGroupName, namespaces: r.Namespaces.Include})
				}
			}
		}
	}
	return out
}

func (r *Router) buildTable(resource string, events map[config.EventType]struct{}, pairings map[config.EventType][]Routes) {
	for evt := range events {
		if _, ok := r.table[resource]; !ok {

			r.table[resource] = []RoutedEvent{{
				event:  evt,
				routes: pairings[evt],
			}}

		} else {
			r.table[resource] = append(r.table[resource], RoutedEvent{event: evt, routes: pairings[evt]})
		}
	}
}

func (r *Router) GetBoundSources(sources config.IndexableMap[config.Sources]) config.IndexableMap[config.Sources] {
	out := make(config.IndexableMap[config.Sources])
	for name, t := range sources {
		if _, ok := r.bindings[name]; ok {
			out[name] = t
		}
	}
	return out
}

type registrationHandler func(resource string) cache.SharedIndexInformer

type eventHandler func(ctx context.Context, resource string, sources []string) func(obj, oldObj interface{})

func (r *Router) RegisterRoutedInformers(targetEvents []config.EventType, handler registrationHandler) {
	resources := r.resourcesForEvents(targetEvents)
	for _, resource := range resources {
		r.informers[resource] = Informer{
			informer:   handler(resource),
			events:     r.resourceEvents(resource),
			log:        r.log,
			mapper:     r.mapper,
			dynamicCli: r.dynamicCli,
		}
	}
}

func (r *Router) RegisterMappedInformer(srcEvent config.EventType, dstEvent config.EventType, handler registrationHandler) {
	srcResources := r.resourcesForEvents([]config.EventType{srcEvent})
	if len(srcResources) == 0 {
		return
	}

	dstResource := "v1/events"
	r.informers[dstResource] = Informer{
		informer:        handler(dstResource),
		events:          []config.EventType{dstEvent},
		mappedResources: srcResources,
		mappedEvent:     srcEvent,
		log:             r.log,
		mapper:          r.mapper,
		dynamicCli:      r.dynamicCli}
}

func (r *Router) HandleRoutedEvent(ctx context.Context, target config.EventType, handlerFn eventHandler) {
	for resource, informer := range r.informers {
		if informer.canHandleEvent(target.String()) {
			sourceRoutes := r.sourceRoutes(resource, target)
			informer.handleRouted(ctx, resource, target, sourceRoutes, handlerFn)
		}
	}
}

func (r *Router) HandleMappedEvent(ctx context.Context, targetEvent config.EventType, handlerFn eventHandler) {
	if informer, ok := r.mappedInformer(targetEvent); ok {
		informer.handleMapped(ctx, targetEvent, r.table, handlerFn)
	}
}

func (r *Router) sourceRoutes(resource string, targetEvent config.EventType) []Routes {
	return sourceRoutes(r.table, resource, targetEvent)
}

func sourceRoutes(routeTable map[string][]RoutedEvent, targetResource string, targetEvent config.EventType) []Routes {
	var out []Routes
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

func (r *Router) mappedInformer(event config.EventType) (Informer, bool) {
	for _, informer := range r.informers {
		if informer.mappedEvent == event {
			return informer, true
		}
	}
	return Informer{}, false
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
