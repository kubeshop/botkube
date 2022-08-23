package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	"github.com/kubeshop/botkube/pkg/sources"
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

	err := c.sourcesRouter.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) (cache.SharedIndexInformer, error) {
		gvr, err := c.parseResourceArg(resource)
		if err != nil {
			c.log.Infof("Unable to parse resource: %s to register with informer\n", resource)
			return nil, err
		}
		return c.dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
	})
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"events": []config.EventType{
				config.CreateEvent,
				config.UpdateEvent,
				config.DeleteEvent,
			},
			"error": err.Error(),
		}).Errorf("Could not register informer.")
		return err
	}

	err = c.sourcesRouter.MapWithEventsInformer(
		config.ErrorEvent,
		config.WarningEvent,
		func(resource string) (cache.SharedIndexInformer, error) {
			gvr, err := c.parseResourceArg(resource)
			if err != nil {
				c.log.Infof("Unable to parse resource: %s to register with informer\n", resource)
				return nil, err
			}
			return c.dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
		})
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"srcEvent": config.ErrorEvent,
			"dstEvent": config.WarningEvent,
			"error":    err.Error(),
		}).Errorf("Could not map event with events informer.")
		return err
	}

	c.sourcesRouter.HandleEvent(
		ctx,
		config.CreateEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.WithFields(logrus.Fields{
					"resource": resource,
					"sources":  sources,
					"event":    config.CreateEvent,
					"object":   obj,
				}).Debugf("Processing K8s resource...")
				c.sendEvent(ctx, obj, resource, config.CreateEvent, sources, nil)
			}
		})

	c.sourcesRouter.HandleEvent(
		ctx,
		config.DeleteEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.WithFields(logrus.Fields{
					"resource": resource,
					"sources":  sources,
					"event":    config.DeleteEvent,
					"object":   obj,
				}).Debugf("Processing K8s resource...")
				c.sendEvent(ctx, obj, resource, config.DeleteEvent, sources, nil)
			}
		})

	c.sourcesRouter.HandleEvent(
		ctx,
		config.UpdateEvent,
		func(ctx context.Context, resource string, sources []string, updateDiffs []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.log.WithFields(logrus.Fields{
					"resource": resource,
					"sources":  sources,
					"event":    config.UpdateEvent,
					"object":   obj,
				}).Debugf("Processing K8s resource...")
				c.sendEvent(ctx, obj, resource, config.UpdateEvent, sources, updateDiffs)
			}
		})

	c.sourcesRouter.HandleMappedEvent(
		ctx,
		config.ErrorEvent,
		func(ctx context.Context, resource string, sources []string, _ []string) func(obj interface{}) {
			return func(obj interface{}) {
				c.sendEvent(ctx, obj, resource, config.ErrorEvent, sources, nil)
			}
		})

	c.log.Info("Starting controller")
	err = sendMessageToNotifiers(ctx, c.notifiers, fmt.Sprintf(controllerStartMsg, c.conf.Settings.ClusterName))
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
		switch {
		case len(sources) == 0 && len(updateDiffs) == 0:
			// skipping least significant update
			c.log.Debug("skipping least significant Update event")
			event.Skip = true
		case len(updateDiffs) > 0:
			event.Messages = append(event.Messages, updateDiffs...)
		default:
			// send event with no diff message
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
