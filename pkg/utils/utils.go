package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"

	log "github.com/infracloudio/botkube/pkg/logging"
	appsV1beta1 "k8s.io/api/apps/v1beta1"
	batchV1 "k8s.io/api/batch/v1"
	apiV1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// RtObjectMap is a map of resource name to respective runtime object
	RtObjectMap map[string]runtime.Object
	// ResourceGetterMap is a map of resource name to resource Getter interface
	ResourceGetterMap map[string]cache.Getter
	// AllowedEventKindsMap is a map to filter valid event kinds
	AllowedEventKindsMap map[EventKind]bool
	// KubeClient is a global kubernetes client to communicate to apiserver
	KubeClient kubernetes.Interface
)

func init() {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		}
		botkubeConf, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Logger.Fatal(err)
		}
		KubeClient, err = kubernetes.NewForConfig(botkubeConf)
		if err != nil {
			log.Logger.Fatal(err)
		}
	} else {
		KubeClient, err = kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			log.Logger.Fatal(err)
		}
	}
	createMaps()
}

// EventKind used in AllowedEventKindsMap to filter event kinds
type EventKind struct {
	Resource  string
	Namespace string
}

func createMaps() {
	botkubeConf, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	RtObjectMap = make(map[string]runtime.Object)
	ResourceGetterMap = make(map[string]cache.Getter)
	AllowedEventKindsMap = make(map[EventKind]bool)

	// Runtime object map
	RtObjectMap["pods"] = &apiV1.Pod{}
	RtObjectMap["nodes"] = &apiV1.Node{}
	RtObjectMap["services"] = &apiV1.Service{}
	RtObjectMap["namespaces"] = &apiV1.Namespace{}
	RtObjectMap["replicationcontrollers"] = &apiV1.ReplicationController{}
	RtObjectMap["persistentvolumes"] = &apiV1.PersistentVolume{}
	RtObjectMap["persistentvolumeclaims"] = &apiV1.PersistentVolumeClaim{}
	RtObjectMap["secrets"] = &apiV1.Secret{}
	RtObjectMap["configmaps"] = &apiV1.ConfigMap{}
	RtObjectMap["deployments"] = &extV1beta1.Deployment{}
	RtObjectMap["daemonsets"] = &extV1beta1.DaemonSet{}
	RtObjectMap["replicasets"] = &extV1beta1.ReplicaSet{}
	RtObjectMap["ingresses"] = &extV1beta1.Ingress{}
	RtObjectMap["jobs"] = &batchV1.Job{}
	RtObjectMap["roles"] = &rbacV1.Role{}
	RtObjectMap["rolebindings"] = &rbacV1.RoleBinding{}
	RtObjectMap["clusterroles"] = &rbacV1.ClusterRole{}
	RtObjectMap["clusterrolebindings"] = &rbacV1.ClusterRoleBinding{}

	// Getter map
	ResourceGetterMap["pods"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["nodes"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["services"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["namespaces"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["replicationcontrollers"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["persistentvolumes"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["persistentvolumeClaim"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["secrets"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["configmaps"] = KubeClient.CoreV1().RESTClient()
	ResourceGetterMap["deployments"] = KubeClient.ExtensionsV1beta1().RESTClient()
	ResourceGetterMap["daemonsets"] = KubeClient.ExtensionsV1beta1().RESTClient()
	ResourceGetterMap["replicasets"] = KubeClient.ExtensionsV1beta1().RESTClient()
	ResourceGetterMap["ingresses"] = KubeClient.ExtensionsV1beta1().RESTClient()
	ResourceGetterMap["jobs"] = KubeClient.BatchV1().RESTClient()
	ResourceGetterMap["roles"] = KubeClient.RbacV1().RESTClient()
	ResourceGetterMap["rolebindings"] = KubeClient.RbacV1().RESTClient()
	ResourceGetterMap["clusterroles"] = KubeClient.RbacV1().RESTClient()
	ResourceGetterMap["clusterrolebindings"] = KubeClient.RbacV1().RESTClient()

	// Allowed event kinds map
	for _, r := range botkubeConf.Resources {
		for _, e := range r.Events {
			if e != config.ErrorEvent {
				continue
			}
			for _, ns := range r.Namespaces {
				if r.Name == "ingresses" {
					AllowedEventKindsMap[EventKind{strings.TrimSuffix(r.Name, "es"), ns}] = true
					continue
				}
				AllowedEventKindsMap[EventKind{strings.TrimSuffix(r.Name, "s"), ns}] = true
			}
		}
	}
	log.Logger.Infof("Allowed Events - %+v", AllowedEventKindsMap)
}

// GetObjectMetaData returns metadata of the given object
func GetObjectMetaData(obj interface{}) metaV1.ObjectMeta {

	var objectMeta metaV1.ObjectMeta

	switch object := obj.(type) {
	case *apiV1.Event:
		objectMeta = object.ObjectMeta
	case *apiV1.Pod:
		objectMeta = object.ObjectMeta
	case *apiV1.Node:
		objectMeta = object.ObjectMeta
	case *apiV1.Namespace:
		objectMeta = object.ObjectMeta
	case *apiV1.PersistentVolume:
		objectMeta = object.ObjectMeta
	case *apiV1.PersistentVolumeClaim:
		objectMeta = object.ObjectMeta
	case *apiV1.ReplicationController:
		objectMeta = object.ObjectMeta
	case *apiV1.Service:
		objectMeta = object.ObjectMeta
	case *apiV1.Secret:
		objectMeta = object.ObjectMeta
	case *apiV1.ConfigMap:
		objectMeta = object.ObjectMeta
	case *extV1beta1.DaemonSet:
		objectMeta = object.ObjectMeta
	case *extV1beta1.Ingress:
		objectMeta = object.ObjectMeta
	case *extV1beta1.ReplicaSet:
		objectMeta = object.ObjectMeta
	case *appsV1beta1.Deployment:
		objectMeta = object.ObjectMeta
	case *batchV1.Job:
		objectMeta = object.ObjectMeta
	case *rbacV1.Role:
		objectMeta = object.ObjectMeta
	case *rbacV1.RoleBinding:
		objectMeta = object.ObjectMeta
	case *rbacV1.ClusterRole:
		objectMeta = object.ObjectMeta
	case *rbacV1.ClusterRoleBinding:
		objectMeta = object.ObjectMeta
	}
	return objectMeta
}

// GetObjectTypeMetaData returns typemetadata of the given object
func GetObjectTypeMetaData(obj interface{}) metaV1.TypeMeta {

	var typeMeta metaV1.TypeMeta

	switch object := obj.(type) {
	case *apiV1.Event:
		typeMeta = object.TypeMeta
	case *apiV1.Pod:
		typeMeta = object.TypeMeta
	case *apiV1.Node:
		typeMeta = object.TypeMeta
	case *apiV1.Namespace:
		typeMeta = object.TypeMeta
	case *apiV1.PersistentVolume:
		typeMeta = object.TypeMeta
	case *apiV1.PersistentVolumeClaim:
		typeMeta = object.TypeMeta
	case *apiV1.ReplicationController:
		typeMeta = object.TypeMeta
	case *apiV1.Service:
		typeMeta = object.TypeMeta
	case *apiV1.Secret:
		typeMeta = object.TypeMeta
	case *apiV1.ConfigMap:
		typeMeta = object.TypeMeta
	case *extV1beta1.DaemonSet:
		typeMeta = object.TypeMeta
	case *extV1beta1.Ingress:
		typeMeta = object.TypeMeta
	case *extV1beta1.ReplicaSet:
		typeMeta = object.TypeMeta
	case *appsV1beta1.Deployment:
		typeMeta = object.TypeMeta
	case *batchV1.Job:
		typeMeta = object.TypeMeta
	case *rbacV1.Role:
		typeMeta = object.TypeMeta
	case *rbacV1.RoleBinding:
		typeMeta = object.TypeMeta
	case *rbacV1.ClusterRole:
		typeMeta = object.TypeMeta
	case *rbacV1.ClusterRoleBinding:
		typeMeta = object.TypeMeta
	}
	return typeMeta
}

// DeleteDoubleWhiteSpace returns slice that removing whitespace from a arg slice
func DeleteDoubleWhiteSpace(slice []string) []string {
	result := []string{}
	for _, s := range slice {
		if s != " " {
			result = append(result, s)
		}
	}
	return result
}
