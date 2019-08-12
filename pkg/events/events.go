package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	appsV1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	apiV1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	rbacV1 "k8s.io/api/rbac/v1"
)

// Level type to store event levels
type Level string

const (
	// Info level
	Info Level = "info"
	// Warn level
	Warn Level = "warn"
	// Debug level
	Debug Level = "debug"
	// Error level
	Error Level = "error"
	// Critical level
	Critical Level = "critical"
)

// Event to store required information from k8s objects
type Event struct {
	Code      string
	Kind      string
	Name      string
	Namespace string
	Messages  []string
	Type      config.EventType
	Reason    string
	Error     string
	Level     Level
	Cluster   string
	TimeStamp time.Time
	Count     int32
	Action    string
	Skip      bool `json:",omitempty"`

	Recommendations []string
}

// LevelMap is a map of event type to Level
var LevelMap map[config.EventType]Level

func init() {
	LevelMap = make(map[config.EventType]Level)
	LevelMap[config.CreateEvent] = Info
	LevelMap[config.UpdateEvent] = Warn
	LevelMap[config.DeleteEvent] = Critical
	LevelMap[config.ErrorEvent] = Error
	LevelMap[config.WarningEvent] = Error
}

// New extract required details from k8s object and returns new Event object
func New(object interface{}, eventType config.EventType, kind string) Event {
	objectTypeMeta := utils.GetObjectTypeMetaData(object)
	objectMeta := utils.GetObjectMetaData(object)

	event := Event{
		Name:      objectMeta.Name,
		Namespace: objectMeta.Namespace,
		Kind:      objectTypeMeta.Kind,
		Level:     LevelMap[eventType],
		Type:      eventType,
	}

	// initialize event.TimeStamp with the time of event creation
	// event.TimeStamp is overwritten later based on the type of the event or
	// kind of the object associated with it
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

	if kind != "events" {
		event.Messages = []string{fmt.Sprintf("Resource %sd\n", eventType.String())}
	}

	switch obj := object.(type) {
	case *apiV1.Event:
		event.Reason = obj.Reason
		event.Messages = append(event.Messages, obj.Message)
		event.Kind = obj.InvolvedObject.Kind
		event.Name = obj.InvolvedObject.Name
		event.Namespace = obj.InvolvedObject.Namespace
		event.Level = LevelMap[config.EventType(strings.ToLower(obj.Type))]
		event.Count = obj.Count
		event.Action = obj.Action
		event.TimeStamp = obj.LastTimestamp.Time
	case *apiV1.Pod:
		event.Kind = "Pod"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *apiV1.Node:
		event.Kind = "Node"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *apiV1.Namespace:
		event.Kind = "Namespace"
	case *apiV1.PersistentVolume:
		event.Kind = "PersistentVolume"
	case *apiV1.PersistentVolumeClaim:
		event.Kind = "PersistentVolumeClaim"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *apiV1.ReplicationController:
		event.Kind = "ReplicationController"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *apiV1.Service:
		event.Kind = "Service"
	case *apiV1.Secret:
		event.Kind = "Secret"
	case *apiV1.ConfigMap:
		event.Kind = "ConfigMap"

	case *extV1beta1.Ingress:
		event.Kind = "Ingress"

	case *appsV1.DaemonSet:
		event.Kind = "DaemonSet"
	case *appsV1.ReplicaSet:
		event.Kind = "ReplicaSet"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *appsV1.Deployment:
		event.Kind = "Deployment"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *appsV1.StatefulSet:
		event.Kind = "StatefulSet"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}

	case *batchV1.Job:
		event.Kind = "Job"
		if eventType == config.UpdateEvent {
			condLen := len(obj.Status.Conditions)
			if condLen != 0 {
				event.TimeStamp = obj.Status.Conditions[condLen-1].LastTransitionTime.Time
			}
		}
	case *rbacV1.Role:
		event.Kind = "Role"
	case *rbacV1.RoleBinding:
		event.Kind = "RoleBinding"
	case *rbacV1.ClusterRole:
		event.Kind = "ClusterRole"
	case *rbacV1.ClusterRoleBinding:
		event.Kind = "ClusterRoleBinding"
	}

	return event
}

// Message returns event message in brief format.
// included as a part of event package to use across handlers.
func (event *Event) Message() (msg string) {
	message := ""
	if len(event.Messages) > 0 {
		for _, m := range event.Messages {
			message = message + m
		}
	}
	if len(event.Recommendations) > 0 {
		recommend := ""
		for _, m := range event.Recommendations {
			recommend = recommend + m
		}
		message = message + fmt.Sprintf("\nRecommendations: %s", recommend)
	}

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		msg = fmt.Sprintf(
			"%s `%s` in of cluster `%s`, namespace `%s` has been %s:\n```%s```",
			event.Kind,
			event.Name,
			event.Cluster,
			event.Namespace,
			event.Type+"d",
			message,
		)
	case config.ErrorEvent:
		msg = fmt.Sprintf(
			"Error Occurred in %s: `%s` of cluster `%s`, namespace `%s`:\n```%s``` ",
			event.Kind,
			event.Name,
			event.Cluster,
			event.Namespace,
			message,
		)
	case config.WarningEvent:
		msg = fmt.Sprintf(
			"Warning %s: `%s` of cluster `%s`, namespace `%s`:\n```%s``` ",
			event.Kind,
			event.Name,
			event.Cluster,
			event.Namespace,
			message,
		)
	}
	return msg
}
