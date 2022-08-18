package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubeshop/botkube/pkg/sources"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
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
	NewForSources(sources map[string]config.Sources, mapKeyOrder []string) recommendation.AggregatedRunner
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
	sourcesRouter         *sources.Router

	dynamicCli dynamic.Interface

	mapper                     meta.RESTMapper
	dynamicKubeInformerFactory dynamicinformer.DynamicSharedInformerFactory
<<<<<<< HEAD
	resourceInformerMap        map[string]cache.SharedIndexInformer
	observedEventKindsMap      map[EventKind]bool
	observedUpdateEventsMap    map[KindNS]config.UpdateSetting
=======
>>>>>>> c706f71 (Removed no longer used code and unnecessary unit tests.)
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
	router *sources.Router,
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
		sourcesRouter:         router,
		reporter:              reporter,
	}
}

// Start creates new informer controllers to watch k8s resources
func (c *Controller) Start(ctx context.Context) error {
	c.dynamicKubeInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.dynamicCli, c.informersResyncPeriod)

	c.sourcesRouter.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) cache.SharedIndexInformer {
		gvr, err := c.parseResourceArg(resource)
		if err != nil {
			c.log.Infof("Unable to parse resource: %s to register with informer\n", resource)
		}
		return c.dynamicKubeInformerFactory.ForResource(gvr).Informer()
	})

	c.sourcesRouter.MapWithEventsInformer(
		config.ErrorEvent,
		config.WarningEvent,
		func(resource string) cache.SharedIndexInformer {
			gvr, err := c.parseResourceArg(resource)
			if err != nil {
				c.log.Infof("Unable to parse resource: %s to register with informer\n", resource)
			}
			return c.dynamicKubeInformerFactory.ForResource(gvr).Informer()
		})

	c.sourcesRouter.HandleEvent(
		context.Background(),
		config.CreateEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.Debugf("Processing add to resource: %q, event: %q, sources: %+v", resource, config.CreateEvent, sources)
				c.sendEvent(ctx, obj, resource, config.CreateEvent, sources, nil)
			}
		})

	c.sourcesRouter.HandleEvent(
		context.Background(),
		config.DeleteEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.Debugf("Processing delete to resource: %q, event: %q, sources: %+v", resource, config.DeleteEvent, sources)
				c.sendEvent(ctx, obj, resource, config.DeleteEvent, sources, nil)
			}
		})

	c.sourcesRouter.HandleEvent(
		context.Background(),
		config.UpdateEvent,
		func(ctx context.Context, resource string, sources []string, updateDiffs []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.Debugf("Processing update to resource: %q, event: %q, sources: %+v\nobject: %+v", resource, config.UpdateEvent, sources, obj)
				c.sendEvent(ctx, obj, resource, config.UpdateEvent, sources, updateDiffs)
			}
		})

	c.sourcesRouter.HandleMappedEvent(
		context.Background(),
		config.ErrorEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.sendEvent(ctx, obj, resource, config.ErrorEvent, sources, nil)
			}
		})

	c.log.Info("Starting controller")
	err := sendMessageToNotifiers(ctx, c.notifiers, fmt.Sprintf(controllerStartMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending first message: %w", err)
	}

	c.startTime = time.Now()

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

func (c *Controller) sendEvent(ctx context.Context, obj interface{}, resource string, eventType config.EventType, sources []string, updateDiffs []string) {
	// Filter namespaces
	objectMeta, err := utils.GetObjectMetaData(ctx, c.dynamicCli, c.mapper, obj)
	if err != nil {
		c.log.Errorf("while getting object metadata: %s", err.Error())
		return
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
		// Send update notification only if fields in updateSetting are changed
		if len(updateDiffs) == 0 {
			// skipping least significant update
			c.log.Debug("skipping least significant Update event")
			event.Skip = true
		} else {
			for _, diff := range updateDiffs {
				event.Messages = append(event.Messages, diff)
			}
		}
	}

	// Filter events
	event = c.filterEngine.Run(ctx, event)
	if event.Skip {
		c.log.Debugf("Skipping event: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		c.log.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	// TODO: Get sources applicable for a given event https://github.com/kubeshop/botkube/issues/676
	// temporary solution - get all sources
	tempSources := c.conf.Sources
	var sourceBindings []string
	for key := range tempSources {
		sourceBindings = append(sourceBindings, key)
	}

	err = c.recommFactory.NewForSources(tempSources, sourceBindings).Do(ctx, &event)
	if err != nil {
		c.log.Errorf("while running recommendations: %w", err)
	}

	// Send event over notifiers
	anonymousEvent := analytics.AnonymizedEventDetailsFrom(event)
	for _, n := range c.notifiers {
		go func(n Notifier) {
			defer analytics.ReportPanicIfOccurs(c.log, c.reporter)

			err := n.SendEvent(ctx, event, sources)
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

//func (c *Controller) shouldSendEvent(namespace string, resource string, eventType config.EventType) bool {
//	eventMap := c.observedEventKindsMap
//	if eventMap == nil {
//		return false
//	}
//
//	if eventMap[EventKind{Resource: resource, Namespace: "all", EventType: eventType}] {
//		return true
//	}
//
//	if eventMap[EventKind{Resource: resource, Namespace: namespace, EventType: eventType}] {
//		return true
//	}
//
//	return false
//}

// TODO: These methods are used only for E2E test purposes. Remove them as a part of https://github.com/kubeshop/botkube/issues/589

// ShouldSendEvent exports Controller functionality for test purposes.
// Deprecated: This is a temporarily exposed part of internal functionality for testing purposes and shouldn't be used in production code.
//func (c *Controller) ShouldSendEvent(namespace string, resource string, eventType config.EventType) bool {
//	return c.shouldSendEvent(namespace, resource, eventType)
//}

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
