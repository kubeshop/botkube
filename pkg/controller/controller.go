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
	"github.com/kubeshop/botkube/pkg/recommendation"
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

// RecommendationFactory defines a factory that creates recommendations.
type RecommendationFactory interface {
	NewForSources(sources map[string]config.Sources) recommendation.Set
}

// Controller watches Kubernetes resources and send events to notifiers.
type Controller struct {
	log                   logrus.FieldLogger
	reporter              AnalyticsReporter
	startTime             time.Time
	conf                  *config.Config
	notifiers             []Notifier
	recommFactory         RecommendationFactory
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
	recommFactory RecommendationFactory,
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
		recommFactory:         recommFactory,
		filterEngine:          filterEngine,
		dynamicCli:            dynamicCli,
		mapper:                mapper,
		informersResyncPeriod: informersResyncPeriod,
		reporter:              reporter,
	}
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

	// Register informers for resource lifecycle events
	if len(c.conf.Sources) > 0 && len(c.conf.Sources.GetFirst().Kubernetes.Resources) > 0 {
		c.log.Info("Registering resource lifecycle informer")
		for _, r := range c.conf.Sources.GetFirst().Kubernetes.Resources {
			if _, ok := c.resourceInformerMap[r.Name]; !ok {
				continue
			}
			c.log.Infof("Adding informer for resource %q", r.Name)
			c.resourceInformerMap[r.Name].AddEventHandler(c.registerEventHandlers(ctx, r.Name, r.Events))
		}
	}

	// Register informers for k8s events
	c.log.Infof("Registering Kubernetes events informer for types %q and %q", config.WarningEvent.String(), config.NormalEvent.String())
	c.dynamicKubeInformerFactory.
		ForResource(schema.GroupVersionResource{Version: "v1", Resource: "events"}).
		Informer().
		AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
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
				switch strings.ToLower(eventObj.Type) {
				case config.WarningEvent.String():
					// Send WarningEvent as ErrorEvents
					c.sendEvent(ctx, obj, nil, utils.GVRToString(gvr), config.ErrorEvent)
				case config.NormalEvent.String():
					// Send NormalEvent as Insignificant InfoEvent
					c.sendEvent(ctx, obj, nil, utils.GVRToString(gvr), config.InfoEvent)
				}
			},
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

func (c *Controller) registerEventHandlers(ctx context.Context, resourceType string, events []config.EventType) (handlerFns cache.ResourceEventHandlerFuncs) {
	for _, event := range events {
		if event == config.AllEvent || event == config.CreateEvent {
			handlerFns.AddFunc = func(obj interface{}) {
				c.log.Debugf("Processing add to %q", resourceType)
				c.sendEvent(ctx, obj, nil, resourceType, config.CreateEvent)
			}
		}

		if event == config.AllEvent || event == config.UpdateEvent {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				c.log.Debugf("Processing update to %q\n Object: %+v\n", resourceType, new)
				c.sendEvent(ctx, new, old, resourceType, config.UpdateEvent)
			}
		}

		if event == config.AllEvent || event == config.DeleteEvent {
			handlerFns.DeleteFunc = func(obj interface{}) {
				c.log.Debugf("Processing delete to %q", resourceType)
				c.sendEvent(ctx, obj, nil, resourceType, config.DeleteEvent)
			}
		}
	}
	return handlerFns
}

func (c *Controller) sendEvent(ctx context.Context, obj, oldObj interface{}, resource string, eventType config.EventType) {
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
		updateSetting, exist := c.observedUpdateEventsMap[KindNS{Resource: resource, Namespace: config.AllNamespaceIndicator}]
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
	event = c.filterEngine.Run(ctx, event)
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

	// TODO: Get sources applicable for a given event https://github.com/kubeshop/botkube/issues/676
	sources := c.conf.Sources // temporary solution - get all sources

	err = c.recommFactory.NewForSources(sources).Run(ctx, &event)
	if err != nil {
		c.log.Errorf("while running recommendations: %w", err)
	}

	// Send event over notifiers
	anonymousEvent := analytics.AnonymizedEventDetailsFrom(event)
	for _, n := range c.notifiers {
		go func(n Notifier) {
			defer analytics.ReportPanicIfOccurs(c.log, c.reporter)

			err := n.SendEvent(ctx, event)
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

	resources := c.conf.Sources.GetFirst().Kubernetes.Resources
	// Create dynamic shared informer factory
	c.dynamicKubeInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.dynamicCli, c.informersResyncPeriod)

	// Init maps
	c.resourceInformerMap = make(map[string]cache.SharedIndexInformer)
	c.observedEventKindsMap = make(map[EventKind]bool)
	c.observedUpdateEventsMap = make(map[KindNS]config.UpdateSetting)

	for _, v := range resources {
		gvr, err := c.parseResourceArg(v.Name)
		if err != nil {
			c.log.Infof("Unable to parse resource: %v\n", v.Name)
			continue
		}

		c.resourceInformerMap[v.Name] = c.dynamicKubeInformerFactory.ForResource(gvr).Informer()
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

	if eventMap[EventKind{Resource: resource, Namespace: config.AllNamespaceIndicator, EventType: eventType}] {
		return true
	}

	if eventMap[EventKind{Resource: resource, Namespace: namespace, EventType: eventType}] {
		return true
	}

	return false
}
