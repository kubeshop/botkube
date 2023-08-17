package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/filterengine"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	pkgConfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	// PluginName is the name of the Kubernetes Botkube plugin.
	PluginName = "kubernetes"

	description = "Consume Kubernetes events and get notifications with additional warnings and recommendations."

	componentLogFieldKey = "component"
)

type RecommendationFactory interface {
	New(cfg config.Config) (recommendation.AggregatedRunner, config.Recommendations)
}

// Source Kubernetes source plugin data structure
type Source struct {
	pluginVersion            string
	config                   config.Config
	logger                   logrus.FieldLogger
	eventCh                  chan source.Event
	startTime                time.Time
	recommFactory            RecommendationFactory
	commandGuard             *command.CommandGuard
	filterEngine             filterengine.FilterEngine
	clusterName              string
	kubeConfig               []byte
	messageBuilder           *MessageBuilder
	isInteractivitySupported bool
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Kubernetes events
func (*Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	if err := pluginx.ValidateKubeConfigProvided(PluginName, input.Context.KubeConfig); err != nil {
		return source.StreamOutput{}, err
	}

	cfg, err := config.MergeConfigs(input.Config)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s := Source{
		startTime: time.Now(),
		eventCh:   make(chan source.Event),
		config:    cfg,
		logger: loggerx.New(pkgConfig.Logger{
			Level: cfg.Log.Level,
		}),
		clusterName:              input.Context.ClusterName,
		kubeConfig:               input.Context.KubeConfig,
		isInteractivitySupported: input.Context.IsInteractivitySupported,
	}

	go consumeEvents(ctx, s)
	return source.StreamOutput{
		Event: s.eventCh,
	}, nil
}

// Metadata returns metadata of Kubernetes configuration
func (s *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     s.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(),
	}, nil
}

func consumeEvents(ctx context.Context, s Source) {
	client, err := NewClient(s.kubeConfig)
	exitOnError(err, s.logger)

	dynamicKubeInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(client.dynamicCli, s.config.InformerResyncPeriod)
	router := NewRouter(client.mapper, client.dynamicCli, s.logger)
	router.BuildTable(&s.config)
	s.recommFactory = recommendation.NewFactory(s.logger.WithField("component", "Recommendations"), client.dynamicCli)
	s.commandGuard = command.NewCommandGuard(s.logger.WithField(componentLogFieldKey, "Command Guard"), client.discoveryCli)
	cmdr := commander.NewCommander(s.logger.WithField(componentLogFieldKey, "Commander"), s.commandGuard, s.config.Commands)
	s.messageBuilder = NewMessageBuilder(s.isInteractivitySupported, s.logger.WithField(componentLogFieldKey, "Message Builder"), cmdr)
	s.filterEngine = filterengine.WithAllFilters(s.logger, client.dynamicCli, client.mapper, s.config.Filters)

	err = router.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) (cache.SharedIndexInformer, error) {
		gvr, err := parseResourceArg(resource, client.mapper)
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

	err = router.MapWithEventsInformer(
		config.ErrorEvent,
		config.WarningEvent,
		func(resource string) (cache.SharedIndexInformer, error) {
			gvr, err := parseResourceArg(resource, client.mapper)
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
		router.RegisterEventHandler(
			ctx,
			s,
			eventType,
			handleEvent,
		)
	}

	router.HandleMappedEvent(
		ctx,
		s,
		config.ErrorEvent,
		handleEvent,
	)

	stopCh := ctx.Done()
	dynamicKubeInformerFactory.Start(stopCh)
}

func handleEvent(ctx context.Context, s Source, e event.Event, updateDiffs []string) {
	s.logger.Debugf("Processing %s to %s/%v in %s namespace", e.Type, e.Resource, e.Name, e.Namespace)
	enrichEventWithAdditionalMetadata(s, &e)

	// Skip older events
	if !e.TimeStamp.IsZero() && e.TimeStamp.Before(s.startTime) {
		s.logger.Debug("Skipping older events")
		return
	}

	// Check for significant Update Events in objects
	if e.Type == config.UpdateEvent && len(updateDiffs) > 0 {
		e.Messages = append(e.Messages, updateDiffs...)
	}

	// Filter events
	e = s.filterEngine.Run(ctx, e)
	if e.Skip {
		s.logger.Debugf("Skipping event: %#v", e)
		return
	}

	if len(e.Kind) <= 0 {
		s.logger.Warn("sendEvent received e with Kind nil. Hence skipping.")
		return
	}

	recRunner, recCfg := s.recommFactory.New(s.config)
	err := recRunner.Do(ctx, &e)
	if err != nil {
		s.logger.Errorf("while running recommendations: %w", err)
		return
	}

	if recommendation.ShouldIgnoreEvent(&recCfg, e) {
		s.logger.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", e)
		return
	}

	msg, err := s.messageBuilder.FromEvent(e, s.config.ExtraButtons)
	if err != nil {
		s.logger.Errorf("while rendering message from event: %w", err)
		return
	}

	message := source.Event{
		Message:         msg,
		RawObject:       e,
		AnalyticsLabels: event.AnonymizedEventDetailsFrom(e),
	}
	s.eventCh <- message
}

func enrichEventWithAdditionalMetadata(s Source, event *event.Event) {
	event.Cluster = s.clusterName
}

func parseResourceArg(arg string, mapper meta.RESTMapper) (schema.GroupVersionResource, error) {
	gvr, err := strToGVR(arg)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("while converting string to GroupVersionReference: %w", err)
	}

	// Validate the GVR provided
	if _, err := mapper.ResourcesFor(gvr); err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvr, nil
}

func strToGVR(arg string) (schema.GroupVersionResource, error) {
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

// jsonSchema contains JSON schema with duplications,
// as some UI components (e.g. https://github.com/rjsf-team/react-jsonschema-form) don't support nested defaults for definitions.
func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
		  "$schema": "http://json-schema.org/draft-07/schema#",
		  "title": "Kubernetes",
		  "description": "%s",
		  "type": "object",
		  "uiSchema": {
			"namespaces": {
			  "include": {
				"ui:classNames": "non-orderable",
				"ui:options": {
				  "orderable": false
				},
				"items": {
				  "ui:options": {
					"label": false
				  }
				}
			  },
			  "exclude": {
				"ui:classNames": "non-orderable",
				"ui:options": {
				  "orderable": false
				},
				"items": {
				  "ui:options": {
					"label": false
				  }
				}
			  }
			},
			"event": {
			  "reason": {
				"include": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				},
				"exclude": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				}
			  },
			  "message": {
				"include": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				},
				"exclude": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				}
			  }
			},
			"annotations": {
			  "ui:classNames": "obj-properties",
			  "additionalProperties": {
				"ui:options": {
				  "label": false
				}
			  }
			},
			"labels": {
			  "ui:classNames": "obj-properties",
			  "additionalProperties": {
				"ui:options": {
				  "label": false
				}
			  }
			},
			"resources": {
			  "ui:classNames": "non-orderable",
			  "ui:options": {
				"orderable": false
			  },
			  "items": {
				"name": {
				  "include": {
					"ui:classNames": "non-orderable",
					"ui:options": {
					  "orderable": false
					},
					"items": {
					  "ui:options": {
						"label": false
					  }
					}
				  },
				  "exclude": {
					"ui:classNames": "non-orderable",
					"ui:options": {
					  "orderable": false
					},
					"items": {
					  "ui:options": {
						"label": false
					  }
					}
				  }
				},
				"namespaces": {
				  "include": {
					"ui:classNames": "non-orderable",
					"ui:options": {
					  "orderable": false
					},
					"items": {
					  "ui:options": {
						"label": false
					  }
					}
				  },
				  "exclude": {
					"ui:classNames": "non-orderable",
					"ui:options": {
					  "orderable": false
					},
					"items": {
					  "ui:options": {
						"label": false
					  }
					}
				  }
				},
				"event": {
				  "reason": {
					"include": {
					  "ui:classNames": "non-orderable",
					  "ui:options": {
						"orderable": false
					  },
					  "items": {
						"ui:options": {
						  "label": false
						}
					  }
					},
					"exclude": {
					  "ui:classNames": "non-orderable",
					  "ui:options": {
						"orderable": false
					  },
					  "items": {
						"ui:options": {
						  "label": false
						}
					  }
					}
				  },
				  "message": {
					"include": {
					  "ui:classNames": "non-orderable",
					  "ui:options": {
						"orderable": false
					  },
					  "items": {
						"ui:options": {
						  "label": false
						}
					  }
					},
					"exclude": {
					  "ui:classNames": "non-orderable",
					  "ui:options": {
						"orderable": false
					  },
					  "items": {
						"ui:options": {
						  "label": false
						}
					  }
					}
				  }
				}
			  },
			  "annotations": {
				"ui:classNames": "obj-properties",
				"additionalProperties": {
				  "ui:options": {
					"label": false
				  }
				}
			  },
			  "labels": {
				"ui:classNames": "obj-properties",
				"additionalProperties": {
				  "ui:options": {
					"label": false
				  }
				}
			  },
			  "updateSetting": {
				"fields": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				}
			  }
			}
		  },
		  "commands": {
			"verbs": {
			  "ui:classNames": "non-orderable",
			  "ui:options": {
				"orderable": false
			  },
			  "items": {
				"ui:options": {
				  "label": false
				}
			  }
			},
			"resources": {
			  "ui:classNames": "non-orderable",
			  "ui:options": {
				"orderable": false
			  },
			  "items": {
				"ui:options": {
				  "label": false
				}
			  }
			}
		  },
		  "additionalProperties": false,
		  "properties": {
			"recommendations": {
			  "title": "Recommendations",
			  "description": "Configure various recommendation insights. If enabled, recommendations work globally for all namespaces.",
			  "type": "object",
			  "properties": {
				"pod": {
				  "title": "Pod Recommendations",
				  "description": "Recommendations for Pod Kubernetes resource.",
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"noLatestImageTag": {
					  "title": "No \"latest\" image tag",
					  "type": "boolean",
					  "description": "If true, notifies about Pod containers that use latest tag for images.",
					  "default": true
					},
					"labelsSet": {
					  "title": "No labels set",
					  "type": "boolean",
					  "description": "If true, notifies about Pod resources created without labels.",
					  "default": true
					}
				  }
				},
				"ingress": {
				  "title": "Ingress Recommendations",
				  "description": "Recommendations for Ingress Kubernetes resource.",
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"backendServiceValid": {
					  "title": "Backend Service valid",
					  "type": "boolean",
					  "description": "If true, notifies about Ingress resources with invalid backend service reference.",
					  "default": true
					},
					"tlsSecretValid": {
					  "title": "TLS Secret valid",
					  "type": "boolean",
					  "description": "If true, notifies about Ingress resources with invalid TLS secret reference.",
					  "default": true
					}
				  }
				}
			  },
			  "additionalProperties": false
			},
			"namespaces": {
			  "description": "Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list.",
			  "$ref": "#/definitions/Namespaces"
			},
			"event": {
			  "$ref": "#/definitions/Event",
			  "default": {
				"types": [
				  "error"
				]
			  },
			  "description": "Describes event constraints for Kubernetes resources. These constraints are applied for every resource specified in the \"Resources\" list, unless they are overridden by the resource's own \"Events\" configuration."
			},
			"annotations": {
			  "description": "Filters Kubernetes resources by annotations. Each resource needs to have all the specified annotations. Regex patterns are not supported.",
			  "$ref": "#/definitions/Annotations"
			},
			"labels": {
			  "$ref": "#/definitions/Labels",
			  "description": "Filters Kubernetes resources by labels. Each resource needs to have all the specified labels. Regex patterns are not supported."
			},
			"resources": {
			  "title": "Resources",
			  "description": "Describes the Kubernetes resources to watch. Each resource can override the namespaces and event configuration. Also, each resource can specify its own 'annotations', 'labels' and 'name' regex.",
			  "type": "array",
			  "default": [
				{
				  "type": "v1/pods"
				},
				{
				  "type": "v1/services"
				},
				{
				  "type": "networking.k8s.io/v1/ingresses"
				},
				{
				  "type": "v1/nodes"
				},
				{
				  "type": "v1/namespaces"
				},
				{
				  "type": "v1/persistentvolumes"
				},
				{
				  "type": "v1/persistentvolumeclaims"
				},
				{
				  "type": "v1/configmaps"
				},
				{
				  "type": "rbac.authorization.k8s.io/v1/roles"
				},
				{
				  "type": "rbac.authorization.k8s.io/v1/rolebindings"
				},
				{
				  "type": "rbac.authorization.k8s.io/v1/clusterrolebindings"
				},
				{
				  "type": "rbac.authorization.k8s.io/v1/clusterroles"
				},
				{
				  "type": "apps/v1/deployments"
				},
				{
				  "type": "apps/v1/statefulsets"
				},
				{
				  "type": "apps/v1/daemonsets"
				},
				{
				  "type": "batch/v1/jobs"
				}
			  ],
			  "items": {
				"title": "Resource",
				"type": "object",
				"required": [
				  "type"
				],
				"properties": {
				  "type": {
					"type": "string",
					"title": "Type",
					"description": "Kubernetes resource type in the format \"{group}/{version}/{kind (plural)}\" format, such as \"apps/v1/deployments\", or \"v1/pods\"."
				  },
				  "namespaces": {
					"description": "Overrides Namespaces defined in global scope for all resources. Describes namespaces for every Kubernetes resources you want to watch or exclude.",
					"title": "Namespaces",
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "include": {
						"title": "Include",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Namespace"
						},
						"description": "List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions: \".*\" - to specify all Namespaces."
					  },
					  "exclude": {
						"title": "Exclude",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Namespace"
						},
						"description": "List of ignored Kubernetes Namespace. It can also contain a regex expressions: \"test-.*\" - to specify all Namespaces."
					  }
					}
				  },
				  "annotations": {
					"description": "Overrides Annotations defined in global scope for all resources. Each resource needs to have all the specified annotations. Regex patterns are not supported.",
					"$ref": "#/definitions/Annotations"
				  },
				  "labels": {
					"description": "Overrides Labels defined in global scope for all resources. Each resource needs to have all the specified annotations. Regex patterns are not supported.",
					"$ref": "#/definitions/Labels"
				  },
				  "name": {
					"title": "Name pattern",
					"description": "Optional patterns to filter events by resource name.",
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "include": {
						"title": "Include",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Reason"
						},
						"description": "List of allowed resource names. It can also contain a regex expressions."
					  },
					  "exclude": {
						"title": "Exclude",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Reason"
						},
						"description": "List of excluded resource names. It can also contain a regex expressions."
					  }
					}
				  },
				  "event": {
					"description": "Overrides Event constraints defined in global scope for all resources.",
					"title": "Event",
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "types": {
						"title": "Types",
						"description": "Lists all event types to be watched.",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Event type",
						  "oneOf": [
							{
							  "const": "create",
							  "title": "Create"
							},
							{
							  "const": "update",
							  "title": "Update"
							},
							{
							  "const": "delete",
							  "title": "Delete"
							},
							{
							  "const": "error",
							  "title": "Error"
							},
							{
							  "const": "warning",
							  "title": "Warning"
							}
						  ]
						},
						"uniqueItems": true
					  },
					  "reason": {
						"title": "Reason",
						"description": "Optional patterns to filter events by event reason.",
						"type": "object",
						"additionalProperties": false,
						"properties": {
						  "include": {
							"title": "Include",
							"type": "array",
							"items": {
							  "type": "string",
							  "title": "Reason"
							},
							"description": "List of allowed event reasons. It can also contain a regex expressions."
						  },
						  "exclude": {
							"title": "Exclude",
							"type": "array",
							"items": {
							  "type": "string",
							  "title": "Reason"
							},
							"description": "List of excluded event reasons. It can also contain a regex expressions."
						  }
						}
					  },
					  "message": {
						"title": "Message",
						"description": "Optional patterns to filter events by message. If a given event has multiple messages, it is considered a match if any of the messages match the regex.",
						"type": "object",
						"additionalProperties": false,
						"properties": {
						  "include": {
							"title": "Include",
							"type": "array",
							"items": {
							  "type": "string",
							  "title": "Message"
							},
							"description": "List of allowed event message patterns."
						  },
						  "exclude": {
							"title": "Exclude",
							"type": "array",
							"items": {
							  "type": "string",
							  "title": "Message"
							},
							"description": "List of excluded event message patterns."
						  }
						}
					  }
					}
				  },
				  "updateSetting": {
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "includeDiff": {
						"title": "Include diff",
						"description": "Includes diff for resource in event notification.",
						"type": "boolean"
					  },
					  "fields": {
						"title": "Fields",
						"description": "Define which properties should be included in the diff. Full JSON field path, such as \"status.phase\", or \"spec.template.spec.containers[*].image\".",
						"type": "array",
						"items": {
						  "type": "string",
						  "title": "Field path"
						}
					  }
					},
					"title": "Update settings",
					"description": "Additional settings for \"Update\" event type."
				  }
				}
			  },
			  "minItems": 1
			},
			"commands": {
			  "title": "Commands",
			  "description": "Configure allowed verbs and resources to display interactive commands on incoming notifications.",
			  "type": "object",
			  "additionalProperties": false,
			  "properties": {
				"verbs": {
				  "type": "array",
				  "title": "Verbs",
				  "description": "Kubectl verbs enabled for interactive notifications.",
				  "default": [
					"api-resources",
					"api-versions",
					"cluster-info",
					"describe",
					"explain",
					"get",
					"logs",
					"top"
				  ],
				  "items": {
					"title": "Verb",
					"type": "string"
				  },
				  "minItems": 0
				},
				"resources": {
				  "type": "array",
				  "title": "Resources",
				  "description": "List of allowed resources for interactive notifications. Each resource must be provided as a plural noun, such as \"deployments\", \"services\" or \"pods\".",
				  "default": [
					"deployments",
					"pods",
					"namespaces",
					"daemonsets",
					"statefulsets",
					"storageclasses",
					"nodes",
					"configmaps",
					"services",
					"ingresses"
				  ],
				  "minItems": 0,
				  "items": {
					"type": "string",
					"title": "Resource"
				  }
				}
			  }
			},
			"filters": {
			  "additionalProperties": false,
			  "title": "Filters",
			  "type": "object",
			  "description": "Configure filters to skip events based on their properties.",
			  "properties": {
				"objectAnnotationChecker": {
				  "type": "boolean",
				  "title": "Object Annotation Checker",
				  "description": "If true, enables support for \"botkube.io/disable\" resource annotation.",
				  "default": true
				},
				"nodeEventsChecker": {
				  "type": "boolean",
				  "title": "Node Events Checker",
				  "description": "If true, filters out Node-related events that are not important.",
				  "default": true
				}
			  }
			},
			"informerResyncPeriod": {
			  "description": "Resync period of Kubernetes informer in a form of a duration string. A duration string is a sequence of decimal numbers, each with optional fraction and a unit suffix, such as \"300ms\", \"1.5h\" or \"2h45m\". Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".",
			  "type": "string",
			  "default": "30m"
			},
			"log": {
			  "title": "Logging",
			  "description": "Logging configuration for the plugin.",
			  "type": "object",
			  "properties": {
				"level": {
				  "title": "Log Level",
				  "description": "Define log level for the plugin. Ensure that Botkube has plugin logging enabled for standard output.",
				  "type": "string",
				  "default": "info",
				  "oneOf": [
					{
					  "const": "panic",
					  "title": "Panic"
					},
					{
					  "const": "fatal",
					  "title": "Fatal"
					},
					{
					  "const": "error",
					  "title": "Error"
					},
					{
					  "const": "warn",
					  "title": "Warning"
					},
					{
					  "const": "info",
					  "title": "Info"
					},
					{
					  "const": "debug",
					  "title": "Debug"
					},
					{
					  "const": "trace",
					  "title": "Trace"
					}
				  ]
				},
				"disableColors": {
				  "type": "boolean",
				  "default": false,
				  "description": "If enabled, disables color logging output.",
				  "title": "Disable Colors"
				}
			  }
			},
            "extraButtons": {
              "title": "Extra Buttons",
              "description": "Extra buttons for actionable items.",
              "type": "object",
              "properties": {
                "enabled": {
                  "type": "boolean",
                  "default": false,
                  "description": "If enabled, renders extra button.",
                  "title": "Enable extra button"
                },
                "trigger": {
                  "title": "Trigger",
                  "description": "Define log level for the plugin. Ensure that Botkube has plugin logging enabled for standard output.",
                  "type": "object",
                  "additionalProperties": false,
                  "properties": {
                    "type": {
                        "title": "Event types",
                        "description": "Event types which will trigger this action",
                        "type": "array",
                        "items": {
                            "type": "string",
                            "title": "Event type"
                        }
                    }
                  }
                },
                "button": {
                  "title": "Button",
                  "description": "Button settings for showing after each matched events.",
                  "type": "object",
                  "additionalProperties": false,
                  "properties": {
                    "commandTpl": {
                        "title": "Command template",
                        "description": "Command template that can be used to generate actual command.",
                        "type": "string"
                    },
                    "displayName": {
                        "title": "Display name",
                        "description": "Display name of this command.",
                        "type": "string"
                    }
                  }
                }
              }
            }
		  },
		  "definitions": {
			"Labels": {
			  "title": "Resource labels",
			  "type": "object",
			  "additionalProperties": {
				"type": "string"
			  }
			},
			"Annotations": {
			  "title": "Resource annotations",
			  "type": "object",
			  "additionalProperties": {
				"type": "string"
			  }
			},
			"Namespaces": {
			  "title": "Namespaces",
			  "type": "object",
			  "additionalProperties": false,
			  "properties": {
				"include": {
				  "title": "Include",
				  "type": "array",
				  "default": [
					".*"
				  ],
				  "items": {
					"type": "string",
					"title": "Namespace"
				  },
				  "minItems": 1,
				  "description": "List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions: \".*\" - to specify all Namespaces."
				},
				"exclude": {
				  "title": "Exclude",
				  "type": "array",
				  "default": [],
				  "items": {
					"type": "string",
					"title": "Namespace"
				  },
				  "description": "List of ignored Kubernetes Namespace. It can also contain a regex expressions: \"test-.*\" - to specify all Namespaces."
				}
			  },
			  "required": [
				"include"
			  ]
			},
			"Event": {
			  "title": "Event",
			  "type": "object",
			  "additionalProperties": false,
			  "required": [
				"types"
			  ],
			  "properties": {
				"types": {
				  "title": "Types",
				  "description": "Lists all event types to be watched.",
				  "type": "array",
				  "items": {
					"type": "string",
					"title": "Event type",
					"oneOf": [
					  {
						"const": "create",
						"title": "Create"
					  },
					  {
						"const": "update",
						"title": "Update"
					  },
					  {
						"const": "delete",
						"title": "Delete"
					  },
					  {
						"const": "error",
						"title": "Error"
					  },
					  {
						"const": "warning",
						"title": "Warning"
					  }
					]
				  },
				  "uniqueItems": true
				},
				"reason": {
				  "title": "Reason",
				  "description": "Optional patterns to filter events by event reason.",
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"include": {
					  "title": "Include",
					  "type": "array",
					  "items": {
						"type": "string",
						"title": "Reason"
					  },
					  "description": "List of allowed event reasons. It can also contain a regex expressions."
					},
					"exclude": {
					  "title": "Exclude",
					  "type": "array",
					  "items": {
						"type": "string",
						"title": "Reason"
					  },
					  "description": "List of excluded event reasons. It can also contain a regex expressions."
					}
				  }
				},
				"message": {
				  "title": "Message",
				  "description": "Optional patterns to filter events by message. If a given event has multiple messages, it is considered a match if any of the messages match the regex.",
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"include": {
					  "title": "Include",
					  "type": "array",
					  "items": {
						"type": "string",
						"title": "Message"
					  },
					  "description": "List of allowed event message patterns."
					},
					"exclude": {
					  "title": "Exclude",
					  "type": "array",
					  "items": {
						"type": "string",
						"title": "Message"
					  },
					  "description": "List of excluded event message patterns."
					}
				  }
				}
			  }
			}
		  }
		}`, description),
	}
}
func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Error(err)
		// Error message is not propagated to Botkube core without this wait.
		time.Sleep(time.Second * 2)
		os.Exit(1)
	}
}
