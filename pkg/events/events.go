package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	appsV1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	networkV1beta1 "k8s.io/api/networking/v1beta1"
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
func New(object interface{}, eventType config.EventType, kind, clusterName string) Event {
	objectTypeMeta := utils.GetObjectTypeMetaData(object)
	objectMeta := utils.GetObjectMetaData(object)

	event := Event{
		Name:      objectMeta.Name,
		Namespace: objectMeta.Namespace,
		Kind:      objectTypeMeta.Kind,
		Level:     LevelMap[eventType],
		Type:      eventType,
		Cluster:   clusterName,
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
			event.Title = strings.Title(fmt.Sprintf("%s %s", kind, eventType.String()))
		default:
			// Events like create, update, delete comes with an extra 'd' at the end
			event.Title = strings.Title(fmt.Sprintf("%s %sd", kind, eventType.String()))
		}
	}

	switch obj := object.(type) {
	case *coreV1.Event:
		event.Reason = obj.Reason
		event.Messages = append(event.Messages, obj.Message)
		event.Kind = obj.InvolvedObject.Kind
		event.Name = obj.InvolvedObject.Name
		event.Namespace = obj.InvolvedObject.Namespace
		event.Level = LevelMap[config.EventType(strings.ToLower(obj.Type))]
		event.Count = obj.Count
		event.Action = obj.Action
		event.TimeStamp = obj.LastTimestamp.Time
	case *coreV1.Pod:
		event.Kind = "Pod"
	case *coreV1.Node:
		event.Kind = "Node"
	case *coreV1.Namespace:
		event.Kind = "Namespace"
	case *coreV1.PersistentVolume:
		event.Kind = "PersistentVolume"
	case *coreV1.PersistentVolumeClaim:
		event.Kind = "PersistentVolumeClaim"
	case *coreV1.ReplicationController:
		event.Kind = "ReplicationController"
	case *coreV1.Service:
		event.Kind = "Service"
	case *coreV1.Secret:
		event.Kind = "Secret"
	case *coreV1.ConfigMap:
		event.Kind = "ConfigMap"

	case *networkV1beta1.Ingress:
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
