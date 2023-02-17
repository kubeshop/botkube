package event

import (
	"fmt"
	"strings"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/pkg/k8sutil"
)

// Event stores data about a given event for Kubernetes object.
//
// WARNING: When adding a new field, check if we shouldn't ignore it when marshalling and sending to ELS.
type Event struct {
	APIVersion      string
	Kind            string
	Code            string
	Title           string
	Name            string
	Namespace       string
	Messages        []string
	Type            config.EventType
	Reason          string
	Error           string
	Level           config.Level
	Cluster         string
	Channel         string
	TimeStamp       time.Time
	Count           int32
	Action          string
	Skip            bool `json:",omitempty"`
	Resource        string
	Recommendations []string
	Warnings        []string
	Actions         []Action

	// The following fields are ignored when marshalling the event by purpose.
	// We send the whole Event struct via sink.Elasticsearch integration.
	// When using ELS dynamic mapping, we should avoid complex, dynamic objects, which could result into type conflicts.
	ObjectMeta metaV1.ObjectMeta `json:"-"`
	Object     interface{}       `json:"-"`
}

// Action describes an automated action for a given event.
type Action struct {
	// Command is the command to be executed, with the bot.CrossPlatformBotName prefix.
	Command          string
	ExecutorBindings []string
	DisplayName      string
}

// HasRecommendationsOrWarnings returns true if event has recommendations or warnings.
func (e *Event) HasRecommendationsOrWarnings() bool {
	return len(e.Recommendations) > 0 || len(e.Warnings) > 0
}

// LevelMap is a map of event type to Level
var LevelMap = map[config.EventType]config.Level{
	config.CreateEvent:  config.Info,
	config.UpdateEvent:  config.Warn,
	config.DeleteEvent:  config.Critical,
	config.ErrorEvent:   config.Error,
	config.WarningEvent: config.Error,
}

// New extract required details from k8s object and returns new Event object
func New(objectMeta metaV1.ObjectMeta, object interface{}, eventType config.EventType, resource string) (Event, error) {
	typeMeta := k8sutil.GetObjectTypeMetaData(object)
	event := Event{
		APIVersion: typeMeta.APIVersion,
		Kind:       typeMeta.Kind,
		ObjectMeta: objectMeta,
		Object:     object,
		Name:       objectMeta.Name,
		Namespace:  objectMeta.Namespace,
		Level:      LevelMap[eventType],
		Type:       eventType,
		Resource:   resource,
	}

	// initialize event.TimeStamp with the time of event creation
	// event.TimeStamp is overwritten later based on the type of the event or
	// resource of the object associated with it
	event.TimeStamp = time.Now()

	// Add TimeStamps
	if eventType == config.CreateEvent {
		event.TimeStamp = objectMeta.CreationTimestamp.Time
	}

	if eventType == config.DeleteEvent {
		if objectMeta.DeletionTimestamp != nil {
			event.TimeStamp = objectMeta.DeletionTimestamp.Time
		}
	}

	switch eventType {
	case config.ErrorEvent, config.InfoEvent:
		event.Title = fmt.Sprintf("%s %s", resource, eventType.String())
	default:
		// Events like create, update, delete comes with an extra 'd' at the end
		event.Title = fmt.Sprintf("%s %sd", resource, eventType.String())
	}

	if typeMeta.Kind == "Event" {
		var eventObj coreV1.Event

		unstrObj, ok := object.(*unstructured.Unstructured)
		if !ok {
			return Event{}, fmt.Errorf("cannot convert type %T into *unstructured.Unstructured", object)
		}

		err := k8sutil.TransformIntoTypedObject(unstrObj, &eventObj)
		if err != nil {
			return Event{}, fmt.Errorf("while transforming object type %T into type: %T: %w", object, eventObj, err)
		}

		event.Reason = eventObj.Reason
		event.Messages = append(event.Messages, eventObj.Message)
		event.Kind = eventObj.InvolvedObject.Kind
		event.APIVersion = eventObj.InvolvedObject.APIVersion
		event.Name = eventObj.InvolvedObject.Name
		event.Namespace = eventObj.InvolvedObject.Namespace
		event.Level = LevelMap[config.EventType(strings.ToLower(eventObj.Type))]
		event.Count = eventObj.Count
		event.Action = eventObj.Action
		event.TimeStamp = eventObj.LastTimestamp.Time
		// Compatible with events.k8s.io/v1
		if eventObj.LastTimestamp.IsZero() && eventObj.Series != nil {
			event.TimeStamp = eventObj.Series.LastObservedTime.Time
			event.Count = eventObj.Series.Count
		}
		if event.TimeStamp.IsZero() {
			// still zero? try event time
			event.TimeStamp = eventObj.EventTime.Time
		}
	}

	return event, nil
}
