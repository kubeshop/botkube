package sources

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

const eventsResource = "v1/events"

type mergedEvents map[string]map[config.EventType]struct{}
type registrationHandler func(resource string) (cache.SharedIndexInformer, error)
type eventHandler func(ctx context.Context, resource string, sources []string, updateDiffs []string) func(obj interface{})

type route struct {
	source        string
	resourceName  string
	labels        map[string]string
	annotations   map[string]string
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

// AddCommunicationsBindings adds source binding from a given communications
func (r *Router) AddCommunicationsBindings(c config.Communications) {
	r.AddBindingsByNameIfConditionTrue(c.Slack.Enabled, c.Slack.Channels)
	r.AddBindingsByNameIfConditionTrue(c.SocketSlack.Enabled, c.SocketSlack.Channels)
	r.AddBindingsByNameIfConditionTrue(c.Mattermost.Enabled, c.Mattermost.Channels)
	r.AddBindingsIfConditionTrue(c.Teams.Enabled, c.Teams.Bindings)
	r.AddBindingsByIDIfConditionTrue(c.Discord.Enabled, c.Discord.Channels)
	r.AddElsIndexSinkBindingsIfConditionTrue(c.Elasticsearch.Enabled, c.Elasticsearch.Indices)

	r.AddSinkBindingsIfConditionTrue(c.Webhook.Enabled, c.Webhook.Bindings)
}

// AddEnabledActionBindings adds source bindings for enabled Actions.
func (r *Router) AddEnabledActionBindings(c config.Actions) {
	for _, act := range c {
		if !act.Enabled {
			continue
		}

		for _, sourceBinding := range act.Bindings.Sources {
			r.bindings[sourceBinding] = struct{}{}
		}
	}
}

// AddBindingsByNameIfConditionTrue adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddBindingsByNameIfConditionTrue(condition bool, c config.IdentifiableMap[config.ChannelBindingsByName]) *Router {
	if !condition {
		return r
	}

	for _, byName := range c {
		r.AddBindings(byName.Bindings)
	}
	return r
}

// AddBindingsByIDIfConditionTrue adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddBindingsByIDIfConditionTrue(condition bool, c config.IdentifiableMap[config.ChannelBindingsByID]) *Router {
	if !condition {
		return r
	}

	for _, byID := range c {
		r.AddBindings(byID.Bindings)
	}
	return r
}

// AddBindingsIfConditionTrue adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddBindingsIfConditionTrue(condition bool, b config.BotBindings) *Router {
	if !condition {
		return r
	}

	return r.AddBindings(b)
}

// AddBindings adds source binding names
// to dictate which source bindings the router should use.
func (r *Router) AddBindings(b config.BotBindings) *Router {
	for _, source := range b.Sources {
		r.bindings[source] = struct{}{}
	}
	return r
}

// AddSinkBindingsIfConditionTrue adds source bindings names
// to dictate which source bindings the router should use.
func (r *Router) AddSinkBindingsIfConditionTrue(condition bool, b config.SinkBindings) *Router {
	if !condition {
		return r
	}

	for _, source := range b.Sources {
		r.bindings[source] = struct{}{}
	}
	return r
}

// AddElsIndexSinkBindingsIfConditionTrue adds source bindings names
// to dictate which source bindings the router should use.
func (r *Router) AddElsIndexSinkBindingsIfConditionTrue(condition bool, b map[string]config.ELSIndex) *Router {
	if !condition {
		return r
	}

	for _, index := range b {
		for _, source := range index.Bindings.Sources {
			r.bindings[source] = struct{}{}
		}
	}
	return r
}

// GetBoundSources returns the Sources the router uses
// based on preconfigured source binding names.
func (r *Router) GetBoundSources(sources map[string]config.Sources) map[string]config.Sources {
	out := make(map[string]config.Sources)
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
		eventRoutes := r.mergeEventRoutes(resource, sources)
		for evt := range resourceEvents {
			r.table[resource] = append(r.table[resource], entry{event: evt, routes: eventRoutes[evt]})
		}
	}
	r.log.Debugf("sources routing table: %+v", r.table)
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

// HandleEvent allows router clients to create handlers that are
// triggered for a target event.
func (r *Router) HandleEvent(ctx context.Context, target config.EventType, handlerFn eventHandler) {
	for resource, informer := range r.registrations {
		if !informer.canHandleEvent(target.String()) {
			continue
		}
		sourceRoutes := r.getSourceRoutes(resource, target)
		informer.handleEvent(ctx, resource, target, sourceRoutes, handlerFn)
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
	return sourceRoutes(r.table, resource, targetEvent)
}

func mergeResourceEvents(sources map[string]config.Sources) mergedEvents {
	out := map[string]map[config.EventType]struct{}{}
	for _, srcGroupCfg := range sources {
		for _, resource := range srcGroupCfg.Kubernetes.Resources {
			if _, ok := out[resource.Type]; !ok {
				out[resource.Type] = make(map[config.EventType]struct{})
			}
			for _, e := range flattenEvents(srcGroupCfg.Kubernetes.Event.Types, resource.Event.Types) {
				out[resource.Type][e] = struct{}{}
			}
		}

		resForRecomms := recommendation.ResourceEventsForConfig(srcGroupCfg.Kubernetes.Recommendations)
		for resourceType, eventType := range resForRecomms {
			if _, ok := out[resourceType]; !ok {
				out[resourceType] = make(map[config.EventType]struct{})
			}
			out[resourceType][eventType] = struct{}{}
		}
	}
	return out
}

func (r *Router) mergeEventRoutes(resource string, sources map[string]config.Sources) map[config.EventType][]route {
	out := make(map[config.EventType][]route)
	for srcGroupName, srcGroupCfg := range sources {
		for _, r := range srcGroupCfg.Kubernetes.Resources {
			for _, e := range flattenEvents(srcGroupCfg.Kubernetes.Event.Types, r.Event.Types) {
				if resource != r.Type {
					continue
				}

				route := route{
					source:      srcGroupName,
					namespaces:  sourceOrResourceNamespaces(srcGroupCfg.Kubernetes.Namespaces, r.Namespaces),
					annotations: sourceOrResourceStringMap(srcGroupCfg.Kubernetes.Annotations, r.Annotations),
					labels:      sourceOrResourceStringMap(srcGroupCfg.Kubernetes.Labels, r.Labels),
					resourceName: r.Name,
				}
				if e == config.UpdateEvent {
					route.updateSetting = config.UpdateSetting{
						Fields:      r.UpdateSetting.Fields,
						IncludeDiff: r.UpdateSetting.IncludeDiff,
					}
				}
				out[e] = append(out[e], route)
			}
		}

		// add routes related to recommendations
		resForRecomms := recommendation.ResourceEventsForConfig(srcGroupCfg.Kubernetes.Recommendations)
		r.setEventRouteForRecommendationsIfShould(&out, resForRecomms, srcGroupName, resource)
	}

	return out
}

func (r *Router) setEventRouteForRecommendationsIfShould(routeMap *map[config.EventType][]route, resForRecomms map[string]config.EventType, srcGroupName, resourceType string) {
	if routeMap == nil {
		r.log.Debug("Skipping setting event route for recommendations as the routeMap is nil")
		return
	}

	eventType, found := resForRecomms[resourceType]
	if !found {
		return
	}

	recommRoute := route{
		source: srcGroupName,
		namespaces: config.Namespaces{
			Include: []string{config.AllNamespaceIndicator},
		},
	}

	// Override route and get all these events for all namespaces.
	// The events without recommendations will be filtered out when sending the event.
	for i, r := range (*routeMap)[eventType] {
		if r.source != srcGroupName {
			continue
		}

		recommRoute.namespaces = r.namespaces
		(*routeMap)[eventType][i] = recommRoute
		return
	}

	// not found, append new route
	(*routeMap)[eventType] = append((*routeMap)[eventType], recommRoute)
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

func flattenEvents(globalEvents []config.EventType, resourceEvents config.KubernetesResourceEventTypes) []config.EventType {
	checkEvents := globalEvents
	if len(resourceEvents) > 0 {
		checkEvents = resourceEvents
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

// sourceOrResourceNamespaces returns the kubernetes source namespaces
// unless the resource namespaces are configured.
func sourceOrResourceNamespaces(sourceNs, resourceNs config.Namespaces) config.Namespaces {
	if resourceNs.IsConfigured() {
		return resourceNs
	}
	return sourceNs
}

func sourceOrResourceStringMap(sourceMap, resourceMap map[string]string) map[string]string {
	if len(resourceMap) > 0 {
		return resourceMap
	}

	return sourceMap
}
