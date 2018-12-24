package events

import (
	"time"

	"github.com/infracloudio/kubeops/pkg/utils"
	appsV1beta1 "k8s.io/api/apps/v1beta1"
	batchV1 "k8s.io/api/batch/v1"
	apiV1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
)

type Level string

const (
	Info     Level = "info"
	Warn     Level = "warn"
	Debug    Level = "debug"
	Error    Level = "error"
	Critical Level = "critical"
)

type Event struct {
	Code            string
	Kind            string
	Name            string
	Namespace       string
	Messages        []string
	Reason          string
	Error           string
	Level           Level
	Cluster         string
	Recommendations []string
	EventTime       time.Time
	FirstTimestamp  time.Time
	LastTimestamp   time.Time
	// The number of times this event has occurred.
	Count int32
	// What action was taken/failed regarding to the Regarding object.
	Action string
}

var LevelMap map[string]Level

func init() {
	LevelMap = make(map[string]Level)
	LevelMap["create"] = Info
	LevelMap["Update"] = Debug
	LevelMap["delete"] = Critical
	LevelMap["error"] = Error
	LevelMap["Warning"] = Critical
	LevelMap["Normal"] = Info
}

func New(object interface{}, eventType string, kind string) Event {
	objectTypeMeta := utils.GetObjectTypeMetaData(object)
	objectMeta := utils.GetObjectMetaData(object)

	event := Event{
		Name:      objectMeta.Name,
		Namespace: objectMeta.Namespace,
		Kind:      objectTypeMeta.Kind,
		Messages:  []string{"Resource " + eventType + "d"},
		Level:     LevelMap[eventType],
	}

	switch obj := object.(type) {
	case *apiV1.Event:
		event.Reason = obj.Reason
		event.Messages = append(event.Messages, obj.Message)
		event.Kind = obj.InvolvedObject.Kind
		event.Name = obj.InvolvedObject.Name
		event.Namespace = obj.InvolvedObject.Namespace
		event.Level = LevelMap[obj.Type]
		event.EventTime = obj.EventTime.Time
		event.FirstTimestamp = obj.FirstTimestamp.Time
		event.LastTimestamp = obj.LastTimestamp.Time
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
	}

	return event
}
