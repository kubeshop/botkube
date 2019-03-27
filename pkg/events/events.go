package events

import (
	"time"

	"github.com/infracloudio/botkube/pkg/utils"
	appsV1beta1 "k8s.io/api/apps/v1beta1"
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
	Type      string
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
var LevelMap map[string]Level

func init() {
	LevelMap = make(map[string]Level)
	LevelMap["create"] = Info
	LevelMap["update"] = Warn
	LevelMap["delete"] = Critical
	LevelMap["error"] = Error
	LevelMap["Warning"] = Critical
	LevelMap["Normal"] = Info
}

// New extract required details from k8s object and returns new Event object
func New(object interface{}, eventType string, kind string) Event {
	objectTypeMeta := utils.GetObjectTypeMetaData(object)
	objectMeta := utils.GetObjectMetaData(object)

	event := Event{
		Name:      objectMeta.Name,
		Namespace: objectMeta.Namespace,
		Kind:      objectTypeMeta.Kind,
		Level:     LevelMap[eventType],
		Type:      eventType,
	}

	// Add TimeStamps
	if eventType == "create" {
		event.TimeStamp = objectMeta.CreationTimestamp.Time
	}

	if eventType == "delete" {
		if objectMeta.DeletionTimestamp != nil {
			event.TimeStamp = objectMeta.DeletionTimestamp.Time
		}
	}

	if kind != "events" {
		event.Messages = []string{"Resource " + eventType + "d\n"}
	}

	switch obj := object.(type) {
	case *apiV1.Event:
		event.Reason = obj.Reason
		event.Messages = append(event.Messages, obj.Message)
		event.Kind = obj.InvolvedObject.Kind
		event.Name = obj.InvolvedObject.Name
		event.Namespace = obj.InvolvedObject.Namespace
		event.Level = LevelMap[obj.Type]
		event.Count = obj.Count
		event.Action = obj.Action
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
	case *extV1beta1.DaemonSet:
		event.Kind = "DaemonSet"
	case *extV1beta1.Ingress:
		event.Kind = "Ingress"
	case *extV1beta1.ReplicaSet:
		event.Kind = "ReplicaSet"
	case *appsV1beta1.Deployment:
		event.Kind = "Deployment"
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
