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
	Title     string
	Kind      string
	Name      string
	Namespace string
	Messages  []string
	Type      config.EventType
	Reason    string
	Error     string
	Level     Level
	Cluster   string
	Channel   string
	TimeStamp time.Time
	Count     int32
	Action    string
	Skip      bool `json:",omitempty"`

	Recommendations []string
	Warnings        []string
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
		switch eventType {
		case config.ErrorEvent, config.InfoEvent:
			event.Title = fmt.Sprintf("Resource %s", eventType.String())
		default:
			// Events like create, update, delete comes with an extra 'd' at the end
			event.Title = fmt.Sprintf("Resource %sd", eventType.String())
		}
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
	case *apiV1.Node:
		event.Kind = "Node"
	case *apiV1.Namespace:
		event.Kind = "Namespace"
	case *apiV1.PersistentVolume:
		event.Kind = "PersistentVolume"
	case *apiV1.PersistentVolumeClaim:
		event.Kind = "PersistentVolumeClaim"
	case *apiV1.ReplicationController:
		event.Kind = "ReplicationController"
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
	case *appsV1.Deployment:
		event.Kind = "Deployment"
	case *appsV1.StatefulSet:
		event.Kind = "StatefulSet"

	case *batchV1.Job:
		event.Kind = "Job"

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
			recommend = recommend + "- " + m
		}
		message = message + fmt.Sprintf("Recommendations:\n%s", recommend)
	}
	if len(event.Warnings) > 0 {
		warning := ""
		for _, m := range event.Warnings {
			warning = warning + "- " + m
		}
		message = message + fmt.Sprintf("Warnings:\n%s", warning)
	}

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"%s *%s/%s* has been %s in *%s* cluster",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"%s *%s/%s* has been %s in *%s* cluster",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		}
	case config.ErrorEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s* in *%s* cluster",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s* in *%s* cluster",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		}
	case config.WarningEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Warning %s: *%s* in *%s* cluster",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Warning %s: *%s* in *%s* cluster",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		}
	}

	// Add message in the attachment if there is any
	if len(message) > 0 {
		msg += fmt.Sprintf("\n```%s```", message)
	}
	return msg
}
