package filters

import (
	"context"
	"fmt"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/infracloudio/botkube/pkg/utils"
)

var (
	serviceGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "services",
	}
	secretGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}
)

// ValidServicePort returns valid Service object if given service with the port exists in the given namespace
func ValidServicePort(ctx context.Context, dynamicCli dynamic.Interface, name, namespace string, port int32) (*coreV1.Service, error) {
	unstructuredService, err := dynamicCli.Resource(serviceGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var serviceObject coreV1.Service
	err = utils.TransformIntoTypedObject(unstructuredService, &serviceObject)
	if err != nil {
		return nil, err
	}
	for _, p := range serviceObject.Spec.Ports {
		if p.Port == port {
			return &serviceObject, nil
		}
	}
	return &serviceObject, fmt.Errorf("Port %d is not exposed by the service %s", port, name)
}

// ValidSecret return Secret object if the secret is present in the specified object
func ValidSecret(ctx context.Context, dynamicCli dynamic.Interface, name, namespace string) (*coreV1.Secret, error) {
	unstructuredSecret, err := dynamicCli.Resource(secretGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var secretObject coreV1.Secret
	err = utils.TransformIntoTypedObject(unstructuredSecret, &secretObject)
	if err != nil {
		return nil, err
	}
	return &secretObject, nil
}

// FindNamespaceFromService returns namespace from fully qualified domain name
func FindNamespaceFromService(service string) string {
	ns := strings.Split(service, ".")
	if len(ns) > 1 {
		return ns[1]
	}
	return "default"
}
