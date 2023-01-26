package kubernetes

import (
	"context"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/recommendation"
	"github.com/sirupsen/logrus"
	"google.golang.org/appengine/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"strings"
)

const (
	// PluginName is the name of the Kubernetes Botkube plugin.
	PluginName = "kubernetes"

	description = "Kubernetes plugin consumes Kubernetes events."
)

// Source Kubernetes source plugin data structure
type Source struct {
	pluginVersion string
	config        Config
	logger        logrus.FieldLogger
	ch            chan []byte
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Kubernetes events
func (s *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	s.ch = make(chan []byte)
	out := source.StreamOutput{Output: s.ch}
	cfg, err := MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s.config = cfg
	s.logger = loggerx.New(loggerx.Config{
		Level: cfg.Log.Level,
	})
	go s.consumeEvents(ctx)
	return out, nil
}

// Metadata returns metadata of Kubernetes configuration
func (s *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     s.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(),
	}, nil
}

func (s *Source) consumeEvents(ctx context.Context) {
	client, err := NewClient(s.config.KubeConfig)
	exitOnError(err, s.logger)

	dynamicKubeInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(client.dynamicCli, s.config.InformerReSyncPeriod)
	sourcesRouter := NewRouter(client.mapper, client.dynamicCli, s.logger)

	err = sourcesRouter.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) (cache.SharedIndexInformer, error) {
		gvr, err := s.parseResourceArg(resource, client.mapper)
		if err != nil {
			s.logger.Infof("Unable to parse resource: %s to register with informer\n", resource)
			return nil, err
		}
		return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
	})
	exitOnError(err, s.logger.WithFields(logrus.Fields{
		"events": []config.EventType{
			config.CreateEvent,
			config.UpdateEvent,
			config.DeleteEvent,
		},
		"error": err.Error(),
	}))

	err = sourcesRouter.MapWithEventsInformer(
		config.ErrorEvent,
		config.WarningEvent,
		func(resource string) (cache.SharedIndexInformer, error) {
			gvr, err := s.parseResourceArg(resource, client.mapper)
			if err != nil {
				s.logger.Infof("Unable to parse resource: %s to register with informer\n", resource)
				return nil, err
			}
			return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
		})
	exitOnError(err, s.logger.WithFields(logrus.Fields{
		"srcEvent": config.ErrorEvent,
		"dstEvent": config.WarningEvent,
		"error":    err.Error(),
	}))

	eventTypes := []config.EventType{
		config.CreateEvent,
		config.DeleteEvent,
		config.UpdateEvent,
	}
	for _, eventType := range eventTypes {
		sourcesRouter.RegisterEventHandler(
			ctx,
			eventType,
			s.handleEvent,
		)
	}

	sourcesRouter.HandleMappedEvent(
		ctx,
		config.ErrorEvent,
		s.handleEvent,
	)

	stopCh := ctx.Done()
	dynamicKubeInformerFactory.Start(stopCh)
}

func (s *Source) handleEvent(ctx context.Context, event event.Event, sources, updateDiffs []string) {
	s.logger.Debugf("Processing %s to %s/%v in %s namespace", event.Type, event.Resource, event.Name, event.Namespace)
	s.enrichEventWithAdditionalMetadata(&event)

	// Skip older events
	if !event.TimeStamp.IsZero() && event.TimeStamp.Before(c.startTime) {
		s.logger.Debug("Skipping older events")
		return
	}

	actions, err := s.actionProvider.RenderedActionsForEvent(event, sources)
	if err != nil {
		s.logger.Errorf("while getting rendered actions for event: %s", err.Error())
		// continue processing event
	}
	event.Actions = actions

	// Check for significant Update Events in objects
	if event.Type == config.UpdateEvent {
		switch {
		case len(sources) == 0 && len(updateDiffs) == 0:
			// skipping least significant update
			s.logger.Debug("skipping least significant Update event")
			event.Skip = true
		case len(updateDiffs) > 0:
			event.Messages = append(event.Messages, updateDiffs...)
		default:
			// send event with no diff message
		}
	}

	// Filter events
	event = s.filterEngine.Run(ctx, event)
	if event.Skip {
		s.logger.Debugf("Skipping event: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		log.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	recRunner, recCfg := c.recommFactory.NewForSources(c.conf.Sources, sources)
	err = recRunner.Do(ctx, &event)
	if err != nil {
		log.Errorf("while running recommendations: %w", err)
	}

	if recommendation.ShouldIgnoreEvent(recCfg, c.conf.Sources, sources, event) {
		log.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", event)
		return
	}
}

func (s *Source) enrichEventWithAdditionalMetadata(event *event.Event) {
	event.Cluster = s.conf.Settings.ClusterName
}

func (s *Source) parseResourceArg(arg string, mapper meta.RESTMapper) (schema.GroupVersionResource, error) {
	gvr, err := s.strToGVR(arg)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("while converting string to GroupVersionReference: %w", err)
	}

	// Validate the GVR provided
	if _, err := mapper.ResourcesFor(gvr); err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvr, nil
}

func (s *Source) strToGVR(arg string) (schema.GroupVersionResource, error) {
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

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "Prometheus",
			"description": "%s",
			"type": "object",
			"properties": {
				"url": {
					"description": "Prometheus endpoint without api version and resource",
					"type": "string",
					"default": "http://localhost:9090",
				},
				"ignoreOldAlerts": {
					"description": "If set as true, Prometheus source plugin will not send alerts that is created before plugin start time",
					"type": "boolean",
					"enum": ["true", "false"],
					"default": true
				},
				"alertStates": {
					"description": "Only the alerts that have state provided in this config will be sent as notification. https://pkg.go.dev/github.com/prometheus/prometheus/rules#AlertState",
					"type": "array",
					"default": ["firing", "pending", "inactive"]
					"enum: ["firing", "pending", "inactive"]
				},
				"log": {
					"description": "Logging configuration",
					"type": "object",
					"properties": {
						"level": {
							"description": "Log level",
							"type": "string",
							"default": "info",
							"enum: ["info", "debug", "error"]
						}
					}
				},
			},
			"required": []
		}`, description),
	}
}
func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
