package kubernetes

import (
	"context"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/internal/loggerx"
	config2 "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"strings"
	"time"
)

const (
	// PluginName is the name of the Kubernetes Botkube plugin.
	PluginName = "kubernetes"

	description = "Kubernetes plugin consumes Kubernetes events."
)

type RecommendationFactory interface {
	NewForSources(cfg config2.Config, mapKeyOrder []string) (recommendation.AggregatedRunner, config.Recommendations)
}

// Source Kubernetes source plugin data structure
type Source struct {
	pluginVersion string
	config        config2.Config
	logger        logrus.FieldLogger
	ch            chan []byte
	startTime     time.Time
	recommFactory RecommendationFactory
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
		startTime:     time.Now(),
	}
}

// Stream streams Kubernetes events
func (s *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	s.ch = make(chan []byte)
	out := source.StreamOutput{Output: s.ch}
	cfg, err := config2.MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s.config = cfg
	s.logger = loggerx.New(loggerx.Config{
		Level: cfg.Log.Level,
	})
	go s.consumeEvents(ctx)
	time.Sleep(time.Minute * 15)
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
	if err != nil {
		exitOnError(err, s.logger.WithFields(logrus.Fields{
			"events": []config.EventType{
				config.CreateEvent,
				config.UpdateEvent,
				config.DeleteEvent,
			},
			"error": err.Error(),
		}))
	}

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
	if err != nil {
		exitOnError(err, s.logger.WithFields(logrus.Fields{
			"srcEvent": config.ErrorEvent,
			"dstEvent": config.WarningEvent,
			"error":    err.Error(),
		}))
	}

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
	s.ch <- []byte("kubernetes event")
	s.logger.Debugf("Processing %s to %s/%v in %s namespace", event.Type, event.Resource, event.Name, event.Namespace)
	s.enrichEventWithAdditionalMetadata(&event)

	// Skip older events
	if !event.TimeStamp.IsZero() && event.TimeStamp.Before(s.startTime) {
		s.logger.Debug("Skipping older events")
		return
	}

	// Check for significant Update Events in objects
	if event.Type == config.UpdateEvent {
		switch {
		case len(sources) == 0 && len(updateDiffs) == 0:
			// skipping the least significant update
			s.logger.Debug("skipping least significant Update event")
			event.Skip = true
		case len(updateDiffs) > 0:
			event.Messages = append(event.Messages, updateDiffs...)
		default:
			// send event with no diff message
		}
	}

	if len(event.Kind) <= 0 {
		s.logger.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	recRunner, recCfg := s.recommFactory.NewForSources(s.config, sources)
	err := recRunner.Do(ctx, &event)
	if err != nil {
		s.logger.Errorf("while running recommendations: %w", err)
	}

	if recommendation.ShouldIgnoreEvent(recCfg, s.config, sources, event) {
		s.logger.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", event)
		return
	}
}

func (s *Source) enrichEventWithAdditionalMetadata(event *event.Event) {
	event.Cluster = s.config.ClusterName
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
		Value: heredoc.Docf(`
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "$ref": "#/definitions/Kubernetes",
  "title": "Kubernetes",
  "description": "%s",
  "definitions": {
    "Kubernetes": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "namespaces": {
          "description": "Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object.",
          "$ref": "#/definitions/KubernetesNamespaces"
        },
        "event": {
          "description": "These constraints are applied for every resource specified in the 'resources' list, unless they are overridden by the resource's own 'events' object.",
          "$ref": "#/definitions/Event"
        },
        "annotations": {
          "description": "Filters Kubernetes resources to watch by annotations.",
          "$ref": "#/definitions/Annotations"
        },
        "labels": {
          "description": "Filters Kubernetes resources to watch by labels.",
          "$ref": "#/definitions/Annotations"
        },
        "resources": {
          "description": "Resources are identified by its type in '{group}/{version}/{kind (plural)}' format. Examples: 'apps/v1/deployments', 'v1/pods'. Each resource can override the namespaces and event configuration by using dedicated 'event' and 'namespaces' field. Also, each resource can specify its own 'annotations', 'labels' and 'name' regex. @default -- See the 'values.yaml' file for full object.",
          "type": "array",
          "items": {
            "$ref": "#/definitions/Resource"
          }
        }
      },
      "title": "Kubernetes"
    },
    "Annotations": {
      "type": "object",
      "additionalProperties": false,
      "title": "Annotations"
    },
    "Event": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "types": {
          "description": "Lists all event types to be watched.",
          "type": "array",
          "items": {
            "$ref": "#/definitions/Type"
          }
        },
        "reason": {
          "description": "Optional regex to filter events by event reason.",
          "type": "string"
        },
        "message": {
          "description": "Optional regex to filter events by message. If a given event has multiple messages, it is considered a match if any of the messages match the regex.",
          "type": "string"
        }
      },
      "title": "Event"
    },
    "KubernetesNamespaces": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "include": {
          "description": "Include contains a list of allowed Namespaces. It can also contain a regex expressions: '- \".*\"' - to specify all Namespaces.",
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "title": "KubernetesNamespaces"
    },
    "Resource": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "type": {
          "type": "string"
        },
        "namespaces": {
          "description": "Overrides kubernetes.namespaces",
          "$ref": "#/definitions/ResourceNamespaces"
        },
        "annotations": {
          "description": "Overrides kubernetes.annotations",
          "$ref": "#/definitions/Annotations"
        },
        "labels": {
          "description": "Overrides kubernetes.labels",
          "$ref": "#/definitions/Annotations"
        },
        "name": {
          "description": "Optional resource name regex.",
          "type": "string"
        },
        "event": {
          "description": "Overrides kubernetes.event",
          "$ref": "#/definitions/Event"
        },
        "updateSetting": {
          "$ref": "#/definitions/UpdateSetting"
        }
      },
      "title": "Resource"
    },
    "ResourceNamespaces": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "include": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "exclude": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "required": [
        "exclude",
        "include"
      ],
      "title": "ResourceNamespaces"
    },
    "UpdateSetting": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "includeDiff": {
          "type": "boolean"
        },
        "fields": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "title": "UpdateSetting"
    },
    "Type": {
      "type": "string",
      "enum": [
        "create",
        "delete",
        "error",
        "update"
      ],
      "title": "Type"
    }
  }
}
`, description),
	}
}
func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
