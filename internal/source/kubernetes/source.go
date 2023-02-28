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

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/filterengine"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

const (
	// PluginName is the name of the Kubernetes Botkube plugin.
	PluginName = "kubernetes"

	description = "Kubernetes plugin consumes Kubernetes events."

	componentLogFieldKey = "component"
)

var emojiForLevel = map[config.Level]string{
	config.Info:     ":large_green_circle:",
	config.Warn:     ":warning:",
	config.Debug:    ":information_source:",
	config.Error:    ":x:",
	config.Critical: ":x:",
}

type RecommendationFactory interface {
	New(cfg config.Config) (recommendation.AggregatedRunner, config.Recommendations)
}

// Source Kubernetes source plugin data structure
type Source struct {
	ctx           context.Context
	pluginVersion string
	config        config.Config
	logger        logrus.FieldLogger
	messageCh     chan source.Message
	startTime     time.Time
	recommFactory RecommendationFactory
	commandGuard  *commander.CommandGuard
	commander     *commander.Commander
	filterEngine  filterengine.FilterEngine
	clusterName   string
	kubeConfig    string
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Kubernetes events
func (*Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	s := Source{}
	s.startTime = time.Now()
	s.messageCh = make(chan source.Message)
	out := source.StreamOutput{Message: s.messageCh}
	cfg, err := config.MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s.config = cfg
	s.logger = loggerx.New(loggerx.Config{
		Level: cfg.Log.Level,
	})
	s.ctx = ctx
	s.clusterName = input.Context.ClusterName
	s.kubeConfig = input.Context.KubeConfig
	go consumeEvents(s)
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

func consumeEvents(s Source) {
	client, err := NewClient(s.kubeConfig)
	exitOnError(err, s.logger)

	dynamicKubeInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(client.dynamicCli, *s.config.InformerReSyncPeriod)
	router := NewRouter(client.mapper, client.dynamicCli, s.logger)
	router.BuildTable(&s.config)
	s.recommFactory = recommendation.NewFactory(s.logger.WithField("component", "Recommendations"), client.dynamicCli)
	s.commandGuard = commander.NewCommandGuard(s.logger.WithField(componentLogFieldKey, "Command Guard"), client.discoveryCli)
	s.commander = commander.NewCommander(s.logger.WithField(componentLogFieldKey, "Commander"), s.commandGuard, s.config.ActionVerbs, s.config.ActionResources)
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
			s,
			eventType,
			handleEvent,
		)
	}

	router.HandleMappedEvent(
		s,
		config.ErrorEvent,
		handleEvent,
	)

	stopCh := s.ctx.Done()
	dynamicKubeInformerFactory.Start(stopCh)
}

func handleEvent(s Source, e event.Event, updateDiffs []string) {
	s.logger.Debugf("Processing %s to %s/%v in %s namespace", e.Type, e.Resource, e.Name, e.Namespace)
	enrichEventWithAdditionalMetadata(s, &e)

	// Skip older events
	if !e.TimeStamp.IsZero() && e.TimeStamp.Before(s.startTime) {
		s.logger.Debug("Skipping older events")
		return
	}

	// Check for significant Update Events in objects
	if e.Type == config.UpdateEvent {
		switch {
		case len(updateDiffs) == 0:
			// skipping the least significant update
			s.logger.Debug("skipping least significant Update e")
			e.Skip = true
		case len(updateDiffs) > 0:
			e.Messages = append(e.Messages, updateDiffs...)
		default:
			// send e with no diff message
		}
	}

	// Filter events
	e = s.filterEngine.Run(s.ctx, e)
	if e.Skip {
		s.logger.Debugf("Skipping e: %#v", e)
		return
	}

	if len(e.Kind) <= 0 {
		s.logger.Warn("sendEvent received e with Kind nil. Hence skipping.")
		return
	}

	recRunner, recCfg := s.recommFactory.New(s.config)
	err := recRunner.Do(s.ctx, &e)
	if err != nil {
		s.logger.Errorf("while running recommendations: %w", err)
	}

	if recommendation.ShouldIgnoreEvent(&recCfg, e) {
		s.logger.Debugf("Skipping e as it is related to recommendation informers and doesn't have any recommendations: %#v", e)
		return
	}

	message := source.Message{
		Data:      messageFrom(s, e),
		Metadata:  e,
		Telemetry: event.AnonymizedEventDetailsFrom(e),
	}
	s.messageCh <- message
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

func messageFrom(s Source, event event.Event, additionalSections ...api.Section) api.Message {
	var sections []api.Section
	section := baseNotificationSection(event)

	// Labels, Messages, Recommendations and Warnings formatted as bullet point lists.
	section.Body.Plaintext = bulletPointEventAttachments(event)

	sections = append(sections, section)
	if len(additionalSections) > 0 {
		sections = append(sections, additionalSections...)
	}

	cmdSection := getInteractiveEventSectionIfShould(s, event)

	if cmdSection != nil {
		sections = append(sections, *cmdSection)
	}
	return api.Message{
		Sections: sections,
	}
}

func getInteractiveEventSectionIfShould(s Source, event event.Event) *api.Section {
	commands, err := s.commander.GetCommandsForEvent(event)
	if err != nil {
		s.logger.Errorf("while getting commands for event: %w", err)
		return nil
	}

	if len(commands) == 0 {
		return nil
	}

	cmdPrefix := fmt.Sprintf("%s kubectl", api.MessageBotNamePlaceholder)
	var optionItems []api.OptionItem
	for _, cmd := range commands {
		optionItems = append(optionItems, api.OptionItem{
			Name:  cmd.Name,
			Value: cmd.Cmd,
		})
	}
	section := interactive.EventCommandsSection(cmdPrefix, optionItems)
	return &section
}

func baseNotificationSection(event event.Event) api.Section {
	emoji := emojiForLevel[event.Level]
	section := api.Section{
		Base: api.Base{
			Header: fmt.Sprintf("%s %s", emoji, event.Title),
		},
	}

	if !event.TimeStamp.IsZero() {
		fallbackTimestampText := event.TimeStamp.Format(time.RFC1123)
		timestampText := fmt.Sprintf("<!date^%d^{date_num} {time_secs}|%s>", event.TimeStamp.Unix(), fallbackTimestampText)
		section.Context = []api.ContextItem{{
			Text: timestampText,
		}}
	}

	return section
}

func bulletPointEventAttachments(event event.Event) string {
	strBuilder := strings.Builder{}
	var labels []string
	appendToListIfNotEmpty(&labels, "Kind", event.Kind)
	appendToListIfNotEmpty(&labels, "Type", event.Type.String())
	appendToListIfNotEmpty(&labels, "Namespace", event.Namespace)
	appendToListIfNotEmpty(&labels, "Name", event.Name)
	appendToListIfNotEmpty(&labels, "Reason", event.Reason)
	appendToListIfNotEmpty(&labels, "Action", event.Action)
	appendToListIfNotEmpty(&labels, "Cluster", event.Cluster)
	writeStringIfNotEmpty(&strBuilder, "Labels", bulletPointListFromMessages(labels))
	writeStringIfNotEmpty(&strBuilder, "Messages", bulletPointListFromMessages(event.Messages))
	writeStringIfNotEmpty(&strBuilder, "Recommendations", bulletPointListFromMessages(event.Recommendations))
	writeStringIfNotEmpty(&strBuilder, "Warnings", bulletPointListFromMessages(event.Warnings))
	return strBuilder.String()
}

func writeStringIfNotEmpty(strBuilder *strings.Builder, title, in string) {
	if in == "" {
		return
	}

	strBuilder.WriteString(fmt.Sprintf("*%s:*\n%s", title, in))
}

func appendToListIfNotEmpty(msgs *[]string, title, in string) {
	if in == "" {
		return
	}

	*msgs = append(*msgs, fmt.Sprintf("%s: %s", title, in))
}

// bulletPointListFromMessages creates a Markdown bullet-point list from messages.
// See https://api.slack.com/reference/surfaces/formatting#block-formatting
func bulletPointListFromMessages(msgs []string) string {
	return joinMessages(msgs, "• ")
}

func joinMessages(msgs []string, msgPrefix string) string {
	if len(msgs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	for _, m := range msgs {
		strBuilder.WriteString(fmt.Sprintf("%s%s\n", msgPrefix, m))
	}

	return strBuilder.String()
}

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`
{
    "$schema": "http://json-schema.org/draft-04/schema#",
    "$ref": "#/definitions/Kubernetes",
    "title": "Kubernetes",
    "description": "%s",
    "definitions":{
    "Kubernetes": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "clusterName": {
          "description": "Cluster name to differentiate incoming messages.",
          "type": "string"
        },
        "informerReSyncPeriod": {
          "description": "Resync period of Kubernetes informer. e.g. 30s",
          "type": "string"
        },
        "log": {
          "description": "Logging configuration.",
          "$ref": "#/definitions/Log"
        },
        "namespaces": {
          "description": "Describes namespaces for every Kubernetes resources you want to watch or exclude. These namespaces are applied to every resource specified in the resources list. However, every specified resource can override this by using its own namespaces object.",
          "$ref": "#/definitions/Namespaces"
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
          "$ref": "#/definitions/Labels"
        },
        "resources": {
          "description": "Resources are identified by its type in '{group}/{version}/{kind (plural)}' format. Examples: 'apps/v1/deployments', 'v1/pods'. Each resource can override the namespaces and event configuration by using dedicated 'event' and 'namespaces' field. Also, each resource can specify its own 'annotations', 'labels' and 'name' regex. @default -- See the 'values.yaml' file for full object.",
          "type": "array",
          "items": {
            "$ref": "#/definitions/Resource"
          }
        },
        "filters": {
          "description": "Filter settings for various sources.",
          "$ref": "#/definitions/Filter"
        },
        "actionVerbs": {
          "description": "Allowed verbs for actionable events.",
          "type": "array",
          "items": {
            "type": "string",
            "enum": [
                "api-resources", 
                "api-versions", 
                "cluster-info", 
                "describe", 
                "explain", 
                "get", 
                "logs", 
                "top"
            ]
          },
          "title": "Action Verbs",
          "uniqueItems": true
        },
		"actionResources": {
          "description": "Allowed resources for actionable events.",
          "type": "array",
          "items": {
            "type": "string",
            "enum": [
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
            ]
          },
          "title": "Action Resources",
          "uniqueItems": true
        }
      },
      "title": "Kubernetes"
    },
    "Annotations": {
      "type": "object",
      "additionalProperties": false,
      "title": "Annotations"
    },
    "Labels": {
      "type": "object",
      "additionalProperties": false,
      "title": "Labels"
    },
    "Event": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "types": {
          "description": "Lists all event types to be watched.",
          "type": "array",
          "items": {
            "type": "string",
            "enum": [
                "create", 
                "update", 
                "delete", 
                "error", 
                "warning", 
                "normal", 
                "info", 
                "all"
            ]
          },
          "title": "Event Types",
          "uniqueItems": true
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
    "Resource": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "type": {
          "type": "string"
        },
        "namespaces": {
          "description": "Overrides kubernetes.namespaces",
          "$ref": "#/definitions/Namespaces"
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
    "Namespaces": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "include": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "List of allowed Kubernetes Namespaces for command execution. It can also contain a regex expressions: \".*\" - to specify all Namespaces."
        },
        "exclude": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "List of ignored Kubernetes Namespace. It can also contain a regex expressions: \"test-.*\" - to specify all Namespaces."
        }
      },
      "required": [
        "exclude",
        "include"
      ],
      "title": "Namespaces"
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
    "Log": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "level": {
          "type": "string",
          "description": "Log level"
        }
      },
      "title": "Log"
    },
    "Recommendations": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "pod": {
            "description": "Recommendations for Pod Kubernetes resource.",
            "$ref": "#/definitions/PodRecommendation"
          },
          "ingress": {
            "description": "Recommendations for Ingress Kubernetes resource.",
            "$ref": "#/definitions/IngressRecommendation"
          }
        },
        "title": "Recommendations"
      },
      "PodRecommendation": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "noLatestImageTag": {
            "type": "boolean",
            "description": "If true, notifies about Pod containers that use latest tag for images."
          },
          "labelsSet": {
            "type": "boolean",
            "description": "If true, notifies about Pod resources created without labels."
          }
        },
        "title": "Pod Recommendations"
      },
      "IngressRecommendation": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "backendServiceValid": {
            "type": "boolean",
            "description": "If true, notifies about Ingress resources with invalid backend service reference."
          },
          "tlsSecretValid": {
            "type": "boolean",
            "description": "If true, notifies about Ingress resources with invalid TLS secret reference."
          }
        },
        "title": "Ingress Recommendations"
      },
      "Filter": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "kubernetes": {
                "description": "Kubernetes filter.",
                "$ref": "#/definitions/KubernetesFilter"
              }
        },
        "title": "Filter"
      },
      "KubernetesFilter": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "objectAnnotationChecker": {
            "type": "boolean",
            "description": "If true, enables support for 'botkube.io/disable' and 'botkube.io/channel' resource annotations."
          },
          "nodeEventsChecker": {
            "type": "boolean",
            "description": "If true, filters out Node-related events that are not important."
          }
        },
        "title": "Kubernetes Filter"
      }
  }
}
`, description),
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
