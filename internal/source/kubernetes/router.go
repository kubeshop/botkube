package kubernetes

import (
	"context"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	pluginConfig "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
)

const eventsResource = "v1/events"

type mergedEvents map[string]map[pluginConfig.EventType]struct{}
type registrationHandler func(resource string) (cache.SharedIndexInformer, error)
type eventHandler func(ctx context.Context, event event.Event, sources []string, updateDiffs []string)

type route struct {
	resourceName  string
	labels        *map[string]string
	annotations   *map[string]string
	namespaces    *pluginConfig.Namespaces
	updateSetting *pluginConfig.UpdateSetting
	event         *pluginConfig.KubernetesEvent
}

func (r route) hasActionableUpdateSetting() bool {
	return len(r.updateSetting.Fields) > 0
}

type entry struct {
	event  pluginConfig.EventType
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

// BuildTable builds the routers routing table marking it ready
// to register, map and handle informer events.
func (r *Router) BuildTable(cfg *pluginConfig.Config) *Router {
	mergedEvents := mergeResourceEvents(cfg)

	for resource, resourceEvents := range mergedEvents {
		eventRoutes := r.mergeEventRoutes(resource, cfg)
		for evt := range resourceEvents {
			r.table[resource] = append(r.table[resource], entry{event: evt, routes: eventRoutes[evt]})
		}
	}
	r.log.Debugf("sources routing table: %+v", r.table)
	return r
}

// RegisterInformers register informers for all the resources that match the target events.
func (r *Router) RegisterInformers(targetEvents []pluginConfig.EventType, handler registrationHandler) error {
	resources := r.resourcesForEvents(targetEvents)
	for _, resource := range resources {
		informer, err := handler(resource)
		if err != nil {
			return err
		}
		r.registrations[resource] = registration{
			informer:   informer,
			events:     r.resourceEvents(resource),
			log:        r.log,
			mapper:     r.mapper,
			dynamicCli: r.dynamicCli,
		}
	}
	return nil
}

// MapWithEventsInformer allows resources to report on an event (srcEvent)
// that can only be observed by watching the general k8s v1/events resource
// using a different event (dstEvent).
//
// For example, you can report the "error" EventType for a resource
// by having the router watch/interrogate the "warning" EventType
// reported by the v1/events resource.
func (r *Router) MapWithEventsInformer(srcEvent pluginConfig.EventType, dstEvent pluginConfig.EventType, handler registrationHandler) error {
	srcResources := r.resourcesForEvents([]pluginConfig.EventType{srcEvent})
	if len(srcResources) == 0 {
		return nil
	}

	informer, err := handler(eventsResource)
	if err != nil {
		return err
	}
	r.registrations[eventsResource] = registration{
		informer:        informer,
		events:          []pluginConfig.EventType{dstEvent},
		mappedResources: srcResources,
		mappedEvent:     srcEvent,
		log:             r.log,
		mapper:          r.mapper,
		dynamicCli:      r.dynamicCli,
	}
	return nil
}

// RegisterEventHandler allows router clients to create handlers that are
// triggered for a target event.
func (r *Router) RegisterEventHandler(ctx context.Context, eventType pluginConfig.EventType, handlerFn eventHandler) {
	for resource, reg := range r.registrations {
		if !reg.canHandleEvent(eventType.String()) {
			continue
		}
		sourceRoutes := r.getSourceRoutes(resource, eventType)
		reg.handleEvent(ctx, resource, eventType, sourceRoutes, handlerFn)
	}
}

// HandleMappedEvent allows router clients to create handlers that are
// triggered for a target mapped event.
func (r *Router) HandleMappedEvent(ctx context.Context, targetEvent pluginConfig.EventType, handlerFn eventHandler) {
	if informer, ok := r.mappedInformer(targetEvent); ok {
		informer.handleMapped(ctx, targetEvent, r.table, handlerFn)
	}
}

// GetSourceRoutes returns all routes for a resource and target event
func (r *Router) getSourceRoutes(resource string, targetEvent pluginConfig.EventType) []route {
	return sourceRoutes(r.table, resource, targetEvent)
}

func mergeResourceEvents(cfg *pluginConfig.Config) mergedEvents {
	out := map[string]map[pluginConfig.EventType]struct{}{}
	for _, resource := range cfg.Resources {
		if _, ok := out[resource.Type]; !ok {
			out[resource.Type] = make(map[pluginConfig.EventType]struct{})
		}
		for _, e := range flattenEventTypes(cfg.Event.Types, resource.Event.Types) {
			out[resource.Type][e] = struct{}{}
		}
	}

	resForRecomms := recommendation.ResourceEventsForConfig(*cfg.Recommendations)
	for resourceType, eventType := range resForRecomms {
		if _, ok := out[resourceType]; !ok {
			out[resourceType] = make(map[pluginConfig.EventType]struct{})
		}
		out[resourceType][eventType] = struct{}{}
	}
	return out
}

func (r *Router) mergeEventRoutes(resource string, cfg *pluginConfig.Config) map[pluginConfig.EventType][]route {
	out := make(map[pluginConfig.EventType][]route)
	for _, r := range cfg.Resources {
		for _, e := range flattenEventTypes(cfg.Event.Types, r.Event.Types) {
			if resource != r.Type {
				continue
			}

			route := route{
				namespaces:   sourceOrResourceNamespaces(cfg.Namespaces, r.Namespaces),
				annotations:  sourceOrResourceStringMap(cfg.Annotations, r.Annotations),
				labels:       sourceOrResourceStringMap(cfg.Labels, r.Labels),
				resourceName: r.Name,
				event:        sourceOrResourceEvent(cfg.Event, r.Event),
			}
			if e == pluginConfig.UpdateEvent {
				route.updateSetting = &pluginConfig.UpdateSetting{
					Fields:      r.UpdateSetting.Fields,
					IncludeDiff: r.UpdateSetting.IncludeDiff,
				}
			}
			out[e] = append(out[e], route)
		}
	}

	// add routes related to recommendations
	resForRecomms := recommendation.ResourceEventsForConfig(*cfg.Recommendations)
	r.setEventRouteForRecommendationsIfShould(&out, resForRecomms, resource)

	return out
}

func (r *Router) setEventRouteForRecommendationsIfShould(routeMap *map[pluginConfig.EventType][]route, resForRecomms map[string]pluginConfig.EventType, resourceType string) {
	if routeMap == nil {
		r.log.Debug("Skipping setting event route for recommendations as the routeMap is nil")
		return
	}

	eventType, found := resForRecomms[resourceType]
	if !found {
		return
	}

	recommRoute := route{
		namespaces: &pluginConfig.Namespaces{
			Include: []string{pluginConfig.AllNamespaceIndicator},
		},
	}

	// Override route and get all these events for all namespaces.
	// The events without recommendations will be filtered out when sending the event.
	for i, r := range (*routeMap)[eventType] {

		recommRoute.namespaces = r.namespaces
		(*routeMap)[eventType][i] = recommRoute
		return
	}

	// not found, append new route
	(*routeMap)[eventType] = append((*routeMap)[eventType], recommRoute)
}

func sourceRoutes(routeTable map[string][]entry, targetResource string, targetEvent pluginConfig.EventType) []route {
	var out []route
	for _, routedEvent := range routeTable[targetResource] {
		if routedEvent.event == targetEvent {
			out = append(out, routedEvent.routes...)
		}
	}
	return out
}

func (r *Router) resourceEvents(resource string) []pluginConfig.EventType {
	var out []pluginConfig.EventType
	for _, routedEvent := range r.table[resource] {
		out = append(out, routedEvent.event)
	}
	return out
}

func (r *Router) resourcesForEvents(targets []pluginConfig.EventType) []string {
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

func (r *Router) mappedInformer(event pluginConfig.EventType) (registration, bool) {
	for _, informer := range r.registrations {
		if informer.mappedEvent == event {
			return informer, true
		}
	}
	return registration{}, false
}

func flattenEventTypes(globalEvents []pluginConfig.EventType, resourceEvents pluginConfig.KubernetesResourceEventTypes) []pluginConfig.EventType {
	checkEvents := globalEvents
	if len(resourceEvents) > 0 {
		checkEvents = resourceEvents
	}

	var out []pluginConfig.EventType
	for _, event := range checkEvents {
		if event == pluginConfig.AllEvent {
			out = append(out, []pluginConfig.EventType{pluginConfig.CreateEvent, pluginConfig.UpdateEvent, pluginConfig.DeleteEvent, pluginConfig.ErrorEvent}...)
		} else {
			out = append(out, event)
		}
	}
	return out
}

// sourceOrResourceNamespaces returns the kubernetes source namespaces
// unless the resource namespaces are configured.
func sourceOrResourceNamespaces(sourceNs *pluginConfig.Namespaces, resourceNs pluginConfig.Namespaces) *pluginConfig.Namespaces {
	if resourceNs.IsConfigured() {
		return &resourceNs
	}
	return sourceNs
}

func sourceOrResourceStringMap(sourceMap *map[string]string, resourceMap map[string]string) *map[string]string {
	if len(resourceMap) > 0 {
		return &resourceMap
	}

	return sourceMap
}

func sourceOrResourceEvent(sourceEvent *pluginConfig.KubernetesEvent, resourceEvent pluginConfig.KubernetesEvent) *pluginConfig.KubernetesEvent {
	if resourceEvent.AreConstraintsDefined() {
		return &resourceEvent
	}

	return sourceEvent
}
