package kubernetes

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/formatx"
)

const eventsResource = "v1/events"

type mergedEvents map[string]map[config.EventType]struct{}
type registrationHandler func(resource string) (cache.SharedIndexInformer, error)
type eventHandler func(ctx context.Context, event event.Event, sources []string, updateDiffs []string)

type route struct {
	Source string

	ResourceName  config.RegexConstraints
	Labels        *map[string]string
	Annotations   *map[string]string
	Namespaces    *config.RegexConstraints
	UpdateSetting *config.UpdateSetting
	Event         *config.KubernetesEvent
}

func (r route) hasActionableUpdateSetting() bool {
	return r.UpdateSetting != nil && len(r.UpdateSetting.Fields) > 0
}

type entry struct {
	Event  config.EventType
	Routes []route
}

// Router maintains handled event types from registered
// informers
type Router struct {
	log           logrus.FieldLogger
	mapper        meta.RESTMapper
	dynamicCli    dynamic.Interface
	table         map[string][]entry
	registrations map[string]registration
}

// NewRouter creates a new router to use for routing event types to registered informers.
func NewRouter(mapper meta.RESTMapper, dynamicCli dynamic.Interface, log logrus.FieldLogger) *Router {
	return &Router{
		log:           log,
		mapper:        mapper,
		dynamicCli:    dynamicCli,
		table:         make(map[string][]entry),
		registrations: make(map[string]registration),
	}
}

// BuildTable builds the routers routing table marking it ready
// to register, map and handle informer events.
func (r *Router) BuildTable(cfgs map[string]SourceConfig) *Router {
	mergedEvents := mergeResourceEvents(cfgs)
	for resource, resourceEvents := range mergedEvents {
		eventRoutes := r.mergeEventRoutes(resource, cfgs)
		for evt := range resourceEvents {
			r.table[resource] = append(r.table[resource], entry{Event: evt, Routes: eventRoutes[evt]})
		}
	}
	r.log.Debug("routing table:", formatx.StructDumper().Sdump(r.table))
	return r
}

// RegisterInformers register informers for all the resources that match the target events.
func (r *Router) RegisterInformers(targetEvents []config.EventType, handler registrationHandler) error {
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
func (r *Router) MapWithEventsInformer(srcEvent config.EventType, dstEvent config.EventType, handler registrationHandler) error {
	srcResources := r.resourcesForEvents([]config.EventType{srcEvent})
	if len(srcResources) == 0 {
		return nil
	}

	informer, err := handler(eventsResource)
	if err != nil {
		return err
	}
	r.registrations[eventsResource] = registration{
		informer:        informer,
		events:          []config.EventType{dstEvent},
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
func (r *Router) RegisterEventHandler(ctx context.Context, eventType config.EventType, handlerFn eventHandler) {
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
func (r *Router) HandleMappedEvent(ctx context.Context, targetEvent config.EventType, handlerFn eventHandler) {
	if informer, ok := r.mappedInformer(targetEvent); ok {
		informer.handleMapped(ctx, targetEvent, r.table, handlerFn)
	}
}

// GetSourceRoutes returns all routes for a resource and target event
func (r *Router) getSourceRoutes(resource string, targetEvent config.EventType) []route {
	return eventRoutes(r.table, resource, targetEvent)
}

func mergeResourceEvents(cfgs map[string]SourceConfig) mergedEvents {
	out := map[string]map[config.EventType]struct{}{}
	for _, srcGroupCfg := range cfgs {
		cfg := srcGroupCfg.cfg
		for _, resource := range cfg.Resources {
			if _, ok := out[resource.Type]; !ok {
				out[resource.Type] = make(map[config.EventType]struct{})
			}
			for _, e := range flattenEventTypes(cfg.Event, resource.Event) {
				out[resource.Type][e] = struct{}{}
			}
		}

		resForRecomms := recommendation.ResourceEventsForConfig(cfg.Recommendations)
		for resourceType, eventType := range resForRecomms {
			if _, ok := out[resourceType]; !ok {
				out[resourceType] = make(map[config.EventType]struct{})
			}
			out[resourceType][eventType] = struct{}{}
		}
	}
	return out
}

func (r *Router) mergeEventRoutes(resource string, cfgs map[string]SourceConfig) map[config.EventType][]route {
	out := make(map[config.EventType][]route)
	for srcGroupName, srcCfg := range cfgs {
		cfg := srcCfg.cfg
		for idx := range cfg.Resources {
			r := cfg.Resources[idx] // make sure that we work on a copy
			for _, e := range flattenEventTypes(cfg.Event, r.Event) {
				if resource != r.Type {
					continue
				}
				route := route{
					Source:       srcGroupName,
					Namespaces:   resourceNamespaces(cfg.Namespaces, &r.Namespaces),
					Annotations:  resourceStringMap(cfg.Annotations, r.Annotations),
					Labels:       resourceStringMap(cfg.Labels, r.Labels),
					ResourceName: r.Name,
					Event:        resourceEvent(cfg.Event, r.Event),
				}
				if e == config.UpdateEvent {
					route.UpdateSetting = &config.UpdateSetting{
						Fields:      r.UpdateSetting.Fields,
						IncludeDiff: r.UpdateSetting.IncludeDiff,
					}
				}

				out[e] = append(out[e], route)
			}
		}
		// add routes related to recommendations
		resForRecomms := recommendation.ResourceEventsForConfig(cfg.Recommendations)
		r.setEventRouteForRecommendationsIfShould(&out, resForRecomms, srcGroupName, resource, &cfg)
	}

	return out
}

func (r *Router) setEventRouteForRecommendationsIfShould(routeMap *map[config.EventType][]route, resForRecomms map[string]config.EventType, srcGroupName, resourceType string, cfg *config.Config) {
	if routeMap == nil {
		r.log.Debug("Skipping setting event route for recommendations as the routeMap is nil")
		return
	}

	eventType, found := resForRecomms[resourceType]
	if !found {
		return
	}

	recommRoute := route{
		Source:     srcGroupName,
		Namespaces: cfg.Namespaces,
		Event: &config.KubernetesEvent{
			Reason:  config.RegexConstraints{},
			Message: config.RegexConstraints{},
			Types:   nil,
		},
	}

	// Override route and get all these events for all namespaces.
	// The events without recommendations will be filtered out when sending the event.
	for i, r := range (*routeMap)[eventType] {
		if r.Source != srcGroupName {
			continue
		}

		recommRoute.Namespaces = resourceNamespaces(cfg.Namespaces, r.Namespaces)
		(*routeMap)[eventType][i] = recommRoute
		return
	}

	// not found, append new route
	(*routeMap)[eventType] = append((*routeMap)[eventType], recommRoute)
}

func eventRoutes(routeTable map[string][]entry, targetResource string, targetEvent config.EventType) []route {
	var out []route
	for _, routedEvent := range routeTable[targetResource] {
		if strings.EqualFold(string(routedEvent.Event), string(targetEvent)) {
			out = append(out, routedEvent.Routes...)
		}
	}
	return out
}

func (r *Router) resourceEvents(resource string) []config.EventType {
	var out []config.EventType
	for _, routedEvent := range r.table[resource] {
		out = append(out, routedEvent.Event)
	}
	return out
}

func (r *Router) resourcesForEvents(targets []config.EventType) []string {
	var out []string
	for _, target := range targets {
		for resource, routedEvents := range r.table {
			for _, routedEvent := range routedEvents {
				if routedEvent.Event == target {
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

func flattenEventTypes(globalEvents *config.KubernetesEvent, resourceEvents config.KubernetesEvent) []config.EventType {
	var checkEvents []config.EventType
	if globalEvents != nil {
		checkEvents = globalEvents.Types
	}
	if len(resourceEvents.Types) > 0 {
		checkEvents = resourceEvents.Types
	}

	var out []config.EventType
	for _, event := range checkEvents {
		if event == config.AllEvent {
			out = append(out, []config.EventType{config.CreateEvent, config.UpdateEvent, config.DeleteEvent, config.ErrorEvent}...)
		} else {
			out = append(out, event)
		}
	}
	return out
}

// resourceNamespaces returns the kubernetes global namespaces
// unless the resource namespaces are configured.
func resourceNamespaces(sourceNs *config.RegexConstraints, resourceNs *config.RegexConstraints) *config.RegexConstraints {
	if resourceNs != nil && resourceNs.AreConstraintsDefined() {
		return resourceNs
	}
	return sourceNs
}

func resourceStringMap(sourceMap *map[string]string, resourceMap map[string]string) *map[string]string {
	if len(resourceMap) > 0 {
		return &resourceMap
	}

	return sourceMap
}

func resourceEvent(sourceEvent *config.KubernetesEvent, resourceEvent config.KubernetesEvent) *config.KubernetesEvent {
	if resourceEvent.AreConstraintsDefined() {
		return &resourceEvent
	}

	return sourceEvent
}
