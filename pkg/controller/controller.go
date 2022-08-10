package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/utils"
)

const (
	controllerStartMsg = "...and now my watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "My watch has ended for cluster '%s'!\nPlease send `@BotKube notifier start` to enable notification once BotKube comes online."
	configUpdateMsg    = "Looks like the configuration is updated for cluster '%s'. I shall halt my watch till I read it."

	finalMessageTimeout = 20 * time.Second
)

// EventKind defines a map key used for event filtering.
// TODO: Do not export it when E2E tests are refactored (https://github.com/kubeshop/botkube/issues/589)
type EventKind struct {
	Resource  string
	Namespace string
	EventType config.EventType
}

// KindNS defines a map key used for update event filtering.
// TODO: Do not export it when E2E tests are refactored (https://github.com/kubeshop/botkube/issues/589)
type KindNS struct {
	Resource  string
	Namespace string
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportHandledEventSuccess reports a successfully handled event using a given communication platform.
	ReportHandledEventSuccess(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails analytics.EventDetails) error

	// ReportHandledEventError reports a failure while handling event using a given communication platform.
	ReportHandledEventError(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails analytics.EventDetails, err error) error

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// Controller watches Kubernetes resources and send events to notifiers.
type Controller struct {
	log                   logrus.FieldLogger
	reporter              AnalyticsReporter
	startTime             time.Time
	conf                  *config.Config
	notifiers             []Notifier
	filterEngine          filterengine.FilterEngine
	informersResyncPeriod time.Duration

	dynamicCli dynamic.Interface
	mapper     meta.RESTMapper

	dynamicKubeInformerFactory dynamicinformer.DynamicSharedInformerFactory
	resourceInformerMap        map[string]cache.SharedIndexInformer
	observedEventKindsMap      map[EventKind]bool
	observedUpdateEventsMap    map[KindNS]config.UpdateSetting
}

// New create a new Controller instance.
func New(log logrus.FieldLogger,
	conf *config.Config,
	notifiers []Notifier,
	filterEngine filterengine.FilterEngine,
	dynamicCli dynamic.Interface,
	mapper meta.RESTMapper,
	informersResyncPeriod time.Duration,
	reporter AnalyticsReporter,
) *Controller {
	return &Controller{
		log:                   log,
		conf:                  conf,
		notifiers:             notifiers,
		filterEngine:          filterEngine,
		dynamicCli:            dynamicCli,
		mapper:                mapper,
		informersResyncPeriod: informersResyncPeriod,
		reporter:              reporter,
	}
}

type eventSources struct {
	event   config.EventType
	sources []string
}

func pivotEventsAndSources(in config.IndexableMap[config.Sources]) map[string][]eventSources {
	out := make(map[string][]eventSources)

	// all unique events per resources
	resourceEvents := map[string]map[config.EventType]struct{}{}
	for _, srcGroupCfg := range in {
		for _, r := range srcGroupCfg.Kubernetes.Resources {
			for _, e := range r.Events {
				if _, ok := resourceEvents[r.Name]; !ok {
					resourceEvents[r.Name] = map[config.EventType]struct{}{e: {}}
				} else {
					resourceEvents[r.Name][e] = struct{}{}
				}
			}
		}
	}

	fmt.Printf("\n\nRESOURCE_EVENTS: %+v\n\n", resourceEvents)

	for resourceName, uniqueEvents := range resourceEvents {

		collectedSources := make(map[config.EventType][]string)

		for srcGroupName, srcGroupCfg := range in {
			for _, r := range srcGroupCfg.Kubernetes.Resources {

				for _, e := range r.Events {
					if resourceName == r.Name {
						collectedSources[e] = append(collectedSources[e], srcGroupName)
					}
				}
			}
		}

		fmt.Printf("\n\nCOLLECTED_SOURCES: %+v\n\n", collectedSources)

		for evt := range uniqueEvents {
			if _, ok := out[resourceName]; !ok {

				out[resourceName] = []eventSources{{
					event:   evt,
					sources: collectedSources[evt],
				}}

			} else {
				out[resourceName] = append(out[resourceName], eventSources{event: evt, sources: collectedSources[evt]})
			}
		}
	}

	return out
}

// Start creates new informer controllers to watch k8s resources
func (c *Controller) Start(ctx context.Context) error {
	c.initInformerMap()

	c.log.Info("Starting controller")
	err := sendMessageToNotifiers(ctx, c.notifiers, fmt.Sprintf(controllerStartMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending first message: %w", err)
	}

	c.startTime = time.Now()

	pivots := pivotEventsAndSources(c.conf.Sources)
	fmt.Printf("\n\nPIVOTS:\n%+v\n\n", pivots)

	// Register informers for resource lifecycle events
	if len(c.conf.Sources) > 0 {
		for resource, pivot := range pivots {
			if _, ok := c.resourceInformerMap[resource]; !ok {
				continue
			}
			fmt.Println("PIVOT", pivot)
			c.resourceInformerMap[resource].AddEventHandler(c.registerEventHandlers(ctx, resource, pivot))
			c.log.Infof("Added informer for resource %q", resource)
		}
	}

	// Register informers for k8s events
	c.log.Infof("Registering Kubernetes events informer for types %q and %q", config.WarningEvent.String(), config.NormalEvent.String())
	createEventsResourceAddHandler := func(cfg map[string][]eventSources) func(obj interface{}) {
		return func(obj interface{}) {
			var eventObj coreV1.Event
			err := utils.TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
			if err != nil {
				c.log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(obj), reflect.TypeOf(eventObj))
			}
			_, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				c.log.Errorf("Failed to get MetaNamespaceKey from event resource")
				return
			}

			// Find involved object type
			gvr, err := utils.GetResourceFromKind(c.mapper, eventObj.InvolvedObject.GroupVersionKind())
			if err != nil {
				c.log.Errorf("Failed to get involved object: %v", err)
				return
			}
			gvrToString := utils.GVRToString(gvr)

			switch strings.ToLower(eventObj.Type) {
			case config.WarningEvent.String():
				// Send WarningEvent as ErrorEvents

				var errorEventSources []string
				eventSourcesList := cfg[gvrToString]
				for _, es := range eventSourcesList {
					if es.event == config.ErrorEvent {
						errorEventSources = es.sources
						break
					}
				}

				c.sendEvent(ctx, obj, nil, gvrToString, config.ErrorEvent, errorEventSources)
			case config.NormalEvent.String():
				// Send NormalEvent as Insignificant InfoEvent
				c.sendEvent(ctx, obj, nil, gvrToString, config.InfoEvent, []string{})
			}
		}
	}
	c.dynamicKubeInformerFactory.
		ForResource(schema.GroupVersionResource{Version: "v1", Resource: "events"}).
		Informer().
		AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: createEventsResourceAddHandler(pivots),
		})

	stopCh := ctx.Done()

	c.dynamicKubeInformerFactory.Start(stopCh)
	<-stopCh

	c.log.Info("Shutdown requested. Sending final message...")
	finalMsgCtx, cancelFn := context.WithTimeout(context.Background(), finalMessageTimeout)
	defer cancelFn()
	err = sendMessageToNotifiers(finalMsgCtx, c.notifiers, fmt.Sprintf(controllerStopMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending final message: %w", err)
	}

	return nil
}

func (c *Controller) registerEventHandlers(ctx context.Context, resource string, eventsWithSources []eventSources) (handlerFns cache.ResourceEventHandlerFuncs) {
	makeCreateFunc := func(sources []string) func(obj interface{}) {
		return func(obj interface{}) {
			c.log.Debugf("Processing add to resource: %q, event: %q, event sources: %q", resource, config.CreateEvent, sources)
			c.sendEvent(ctx, obj, nil, resource, config.CreateEvent, sources)
		}
	}

	makeDeleteFunc := func(sources []string) func(obj interface{}) {
		return func(obj interface{}) {
			c.log.Debugf("Processing delete to resource: %q, event: %q, event sources: %q", resource, config.DeleteEvent, sources)
			c.sendEvent(ctx, obj, nil, resource, config.DeleteEvent, sources)
		}
	}

	for _, es := range eventsWithSources {
		event := es.event
		fmt.Println("event: ", event)
		if event == config.AllEvent || event == config.CreateEvent {
			handlerFns.AddFunc = makeCreateFunc(es.sources)
		}

		if event == config.AllEvent || event == config.UpdateEvent {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				c.log.Debugf("Processing update to %q\n Object: %+v\n", resource, new)
				c.sendEvent(ctx, new, old, resource, config.UpdateEvent, es.sources)
			}
		}

		if event == config.AllEvent || event == config.DeleteEvent {
			handlerFns.DeleteFunc = makeDeleteFunc(es.sources)
		}
	}
	return handlerFns
}

//func (c *Controller) sendEvent(ctx context.Context, obj, oldObj interface{}, source string, resource string, eventType config.EventType) {
func (c *Controller) sendEvent(ctx context.Context, obj, oldObj interface{}, resource string, eventType config.EventType, eventSources []string) {
	// Filter namespaces
	objectMeta, err := utils.GetObjectMetaData(ctx, c.dynamicCli, c.mapper, obj)
	if err != nil {
		c.log.Errorf("while getting object metadata: %s", err.Error())
		return
	}

	switch eventType {
	case config.InfoEvent:
		// Skip if ErrorEvent is not configured for the resource
		if !c.shouldSendEvent(objectMeta.Namespace, resource, config.ErrorEvent) {
			c.log.Debugf("Ignoring %q to %s/%v in %q namespace", eventType, resource, objectMeta.Name, objectMeta.Namespace)
			return
		}
	default:
		if !c.shouldSendEvent(objectMeta.Namespace, resource, eventType) {
			c.log.Debugf("Ignoring %q to %s/%v in %q namespace", eventType, resource, objectMeta.Name, objectMeta.Namespace)
			return
		}
	}

	c.log.Debugf("Processing %s to %s/%v in %s namespace", eventType, resource, objectMeta.Name, objectMeta.Namespace)

	// Create new event object
	event, err := events.New(objectMeta, obj, eventType, resource, c.conf.Settings.ClusterName)
	if err != nil {
		c.log.Errorf("while creating new event: %w", err)
		return
	}

	// Skip older events
	if !event.TimeStamp.IsZero() {
		if event.TimeStamp.Before(c.startTime) {
			c.log.Debug("Skipping older events")
			return
		}
	}

	// Check for significant Update Events in objects
	if eventType == config.UpdateEvent {
		var updateMsg string
		// Check if all namespaces allowed
		updateSetting, exist := c.observedUpdateEventsMap[KindNS{Resource: resource, Namespace: "all"}]
		if !exist {
			// Check if specified namespace is allowed
			updateSetting, exist = c.observedUpdateEventsMap[KindNS{Resource: resource, Namespace: objectMeta.Namespace}]
		}
		if exist {
			// Calculate object diff as per the updateSettings
			var oldUnstruct, newUnstruct *unstructured.Unstructured
			var ok bool
			if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
				c.log.Errorf("Failed to typecast object to Unstructured. Skipping event: %#v", event)
			}
			if newUnstruct, ok = obj.(*unstructured.Unstructured); !ok {
				c.log.Errorf("Failed to typecast object to Unstructured. Skipping event: %#v", event)
			}
			updateMsg, err = utils.Diff(oldUnstruct.Object, newUnstruct.Object, updateSetting)
			if err != nil {
				c.log.Errorf("while getting diff: %w", err)
			}
		}

		// Send update notification only if fields in updateSetting are changed
		if len(updateMsg) > 0 {
			if updateSetting.IncludeDiff {
				event.Messages = append(event.Messages, updateMsg)
			}
		} else {
			// skipping least significant update
			c.log.Debug("skipping least significant Update event")
			event.Skip = true
		}
	}

	// Filter events
	event = c.filterEngine.Run(ctx, obj, event)
	if event.Skip {
		c.log.Debugf("Skipping event: %#v", event)
		return
	}

	// Skip unpromoted insignificant InfoEvents
	if event.Type == config.InfoEvent {
		c.log.Debugf("Skipping Insignificant InfoEvent: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		c.log.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	// check if Recommendations are disabled
	if !c.conf.Sources.GetFirst().Recommendations {
		event.Recommendations = nil
		c.log.Debug("Skipping Recommendations in Event Notifications")
	}

	// Send event over notifiers
	anonymousEvent := analytics.AnonymizedEventDetailsFrom(event)
	for _, n := range c.notifiers {
		go func(n Notifier) {
			defer analytics.ReportPanicIfOccurs(c.log, c.reporter)

			err := n.SendEvent(ctx, event, eventSources)
			if err != nil {
				reportErr := c.reporter.ReportHandledEventError(n.Type(), n.IntegrationName(), anonymousEvent, err)
				if reportErr != nil {
					err = multierror.Append(err, fmt.Errorf("while reporting analytics: %w", reportErr))
				}

				c.log.Errorf("while sending event: %s", err.Error())
			}

			reportErr := c.reporter.ReportHandledEventSuccess(n.Type(), n.IntegrationName(), anonymousEvent)
			if reportErr != nil {
				c.log.Errorf("while reporting analytics: %w", err)
			}
		}(n)
	}
}

func (c *Controller) initInformerMap() {
	if len(c.conf.Sources) == 0 {
		return
	}

	c.dynamicKubeInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.dynamicCli, c.informersResyncPeriod)

	// Init maps
	c.resourceInformerMap = make(map[string]cache.SharedIndexInformer)
	c.observedEventKindsMap = make(map[EventKind]bool)
	c.observedUpdateEventsMap = make(map[KindNS]config.UpdateSetting)

	for srcGroupName, srcGroupCfg := range c.conf.Sources {
		resources := srcGroupCfg.Kubernetes.Resources

		for _, r := range resources {
			if _, ok := c.resourceInformerMap[r.Name]; ok {
				continue
			}

			gvr, err := c.parseResourceArg(r.Name)
			if err != nil {
				c.log.Infof("Unable to parse resource: %s for source: %s\n", r.Name, srcGroupName)
				continue
			}

			c.resourceInformerMap[r.Name] = c.dynamicKubeInformerFactory.ForResource(gvr).Informer()
		}

		// Allowed event kinds map and Allowed Update Events Map
		for _, r := range resources {
			allEvents := false
			for _, e := range r.Events {
				if e == config.AllEvent {
					allEvents = true
					break
				}
				for _, ns := range r.Namespaces.Include {
					c.observedEventKindsMap[EventKind{Resource: r.Name, Namespace: ns, EventType: e}] = true
				}
				// AllowedUpdateEventsMap entry is created only for UpdateEvent
				if e == config.UpdateEvent {
					for _, ns := range r.Namespaces.Include {
						c.observedUpdateEventsMap[KindNS{Resource: r.Name, Namespace: ns}] = r.UpdateSetting
					}
				}
			}

			// For AllEvent type, add all events to map
			if allEvents {
				events := []config.EventType{config.CreateEvent, config.UpdateEvent, config.DeleteEvent, config.ErrorEvent}
				for _, ev := range events {
					for _, ns := range r.Namespaces.Include {
						c.observedEventKindsMap[EventKind{Resource: r.Name, Namespace: ns, EventType: ev}] = true
						c.observedUpdateEventsMap[KindNS{Resource: r.Name, Namespace: ns}] = r.UpdateSetting
					}
				}
			}
		}
	}

	c.log.Infof("Allowed Events: %+v", c.observedEventKindsMap)
	c.log.Infof("Allowed UpdateEvents: %+v", c.observedUpdateEventsMap)
}

func (c *Controller) parseResourceArg(arg string) (schema.GroupVersionResource, error) {
	gvr, err := c.strToGVR(arg)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("while converting string to GroupVersionReference: %w", err)
	}

	// Validate the GVR provided
	if _, err := c.mapper.ResourcesFor(gvr); err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvr, nil
}

func (c *Controller) strToGVR(arg string) (schema.GroupVersionResource, error) {
	const separator = "/"
	gvrStrParts := strings.Split(arg, separator)
	switch len(gvrStrParts) {
	case 2:
		return schema.GroupVersionResource{Group: "", Version: gvrStrParts[0], Resource: gvrStrParts[1]}, nil
	case 3:
		return schema.GroupVersionResource{Group: gvrStrParts[0], Version: gvrStrParts[1], Resource: gvrStrParts[2]}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("invalid string: expected 2 or 3 parts when split by %q", separator)
	}
}

func (c *Controller) shouldSendEvent(namespace string, resource string, eventType config.EventType) bool {
	eventMap := c.observedEventKindsMap
	if eventMap == nil {
		return false
	}

	if eventMap[EventKind{Resource: resource, Namespace: "all", EventType: eventType}] {
		return true
	}

	if eventMap[EventKind{Resource: resource, Namespace: namespace, EventType: eventType}] {
		return true
	}

	return false
}

// TODO: These methods are used only for E2E test purposes. Remove them as a part of https://github.com/kubeshop/botkube/issues/589

// ShouldSendEvent exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
func (c *Controller) ShouldSendEvent(namespace string, resource string, eventType config.EventType) bool {
	return c.shouldSendEvent(namespace, resource, eventType)
}

// ObservedEventKindsMap exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
func (c *Controller) ObservedEventKindsMap() map[EventKind]bool {
	return c.observedEventKindsMap
}

// SetObservedEventKindsMap exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
func (c *Controller) SetObservedEventKindsMap(observedEventKindsMap map[EventKind]bool) {
	c.observedEventKindsMap = observedEventKindsMap
}

// ObservedUpdateEventsMap exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
func (c *Controller) ObservedUpdateEventsMap() map[KindNS]config.UpdateSetting {
	return c.observedUpdateEventsMap
}

// SetObservedUpdateEventsMap exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
func (c *Controller) SetObservedUpdateEventsMap(observedUpdateEventsMap map[KindNS]config.UpdateSetting) {
	c.observedUpdateEventsMap = observedUpdateEventsMap
}
