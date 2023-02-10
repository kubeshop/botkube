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
	"github.com/kubeshop/botkube/internal/source"
	"github.com/kubeshop/botkube/internal/status"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

const (
	controllerStartMsg = "My watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "My watch has ended for cluster '%s'. See you soon!"

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
	NewForSources(sources map[string]config.Sources, mapKeyOrder []string) (recommendation.AggregatedRunner, config.Recommendations)
}

// ActionProvider defines a provider that is responsible for automated actions.
type ActionProvider interface {
	RenderedActionsForEvent(event event.Event, sourceBindings []string) ([]event.Action, error)
	ExecuteEventAction(ctx context.Context, action event.Action) interactive.CoreMessage
}

// Controller watches Kubernetes resources and send events to notifiers.
type Controller struct {
	log                   logrus.FieldLogger
	reporter              AnalyticsReporter
	startTime             time.Time
	conf                  *config.Config
	notifiers             []notifier.Notifier
	recommFactory         RecommendationFactory
	filterEngine          filterengine.FilterEngine
	informersResyncPeriod time.Duration
	sourcesRouter         *source.Router
	actionProvider        ActionProvider
	statusReporter        status.StatusReporter

	dynamicCli dynamic.Interface

	mapper                     meta.RESTMapper
	dynamicKubeInformerFactory dynamicinformer.DynamicSharedInformerFactory
}

// New create a new Controller instance.
func New(log logrus.FieldLogger,
	conf *config.Config,
	notifiers []notifier.Notifier,
	recommFactory RecommendationFactory,
	filterEngine filterengine.FilterEngine,
	dynamicCli dynamic.Interface,
	mapper meta.RESTMapper,
	informersResyncPeriod time.Duration,
	router *source.Router,
	actionProvider ActionProvider,
	reporter AnalyticsReporter,
	statusReporter status.StatusReporter,
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
		actionProvider:        actionProvider,
		reporter:              reporter,
		statusReporter:        statusReporter,
	}
}

// Start creates new informer controllers to watch k8s resources
func (c *Controller) Start(ctx context.Context) error {
	c.log.Info("Starting controller...")
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

	eventTypes := []config.EventType{
		config.CreateEvent,
		config.DeleteEvent,
		config.UpdateEvent,
	}
	for _, eventType := range eventTypes {
		c.sourcesRouter.RegisterEventHandler(
			ctx,
			eventType,
			c.handleEvent,
		)
	}

	c.sourcesRouter.HandleMappedEvent(
		ctx,
		config.ErrorEvent,
		c.handleEvent,
	)

	c.log.Info("Sending welcome message...")
	err = notifier.SendPlaintextMessage(ctx, c.notifiers, fmt.Sprintf(controllerStartMsg, c.conf.Settings.ClusterName))
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
	err = notifier.SendPlaintextMessage(finalMsgCtx, c.notifiers, fmt.Sprintf(controllerStopMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending final message: %w", err)
	}

	// use separate ctx as parent ctx is already cancelled
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := c.statusReporter.ReportDeploymentShutdown(ctxTimeout); err != nil {
		return fmt.Errorf("while reporting botkube shutdown: %w", err)
	}

	return nil
}

func (c *Controller) handleEvent(ctx context.Context, event event.Event, sources, updateDiffs []string) {
	c.log.Debugf("Processing %s to %s/%v in %s namespace", event.Type, event.Resource, event.Name, event.Namespace)
	c.enrichEventWithAdditionalMetadata(&event)

	// Skip older events
	if !event.TimeStamp.IsZero() && event.TimeStamp.Before(c.startTime) {
		c.log.Debug("Skipping older events")
		return
	}

	actions, err := c.actionProvider.RenderedActionsForEvent(event, sources)
	if err != nil {
		c.log.Errorf("while getting rendered actions for event: %s", err.Error())
		// continue processing event
	}
	event.Actions = actions

	// Check for significant Update Events in objects
	if event.Type == config.UpdateEvent {
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

	recRunner, recCfg := c.recommFactory.NewForSources(c.conf.Sources, sources)
	err = recRunner.Do(ctx, &event)
	if err != nil {
		c.log.Errorf("while running recommendations: %w", err)
	}

	if recommendation.ShouldIgnoreEvent(recCfg, c.conf.Sources, sources, event) {
		c.log.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", event)
		return
	}

	// Send event over notifiers
	anonymousEvent := analytics.AnonymizedEventDetailsFrom(event)
	for _, n := range c.notifiers {
		go func(n notifier.Notifier) {
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

	// execute actions
	for _, action := range event.Actions {
		c.log.Infof("Executing action %q (command: %q)...", action.DisplayName, action.Command)
		genericMsg := c.actionProvider.ExecuteEventAction(ctx, action)
		for _, n := range c.notifiers {
			go func(n notifier.Notifier) {
				defer analytics.ReportPanicIfOccurs(c.log, c.reporter)
				err := n.SendMessage(ctx, genericMsg, sources)
				if err != nil {
					c.log.Errorf("while sending event: %s", err.Error())
				}
			}(n)
		}
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

func (c *Controller) enrichEventWithAdditionalMetadata(event *event.Event) {
	event.Cluster = c.conf.Settings.ClusterName
}
