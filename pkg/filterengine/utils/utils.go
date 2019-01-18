package utils

import (
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/utils"
	apiV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidService returns Service object is service given service exists in the given namespace
func ValidService(name, namespace string) (*apiV1.Service, error) {
	serviceClient := utils.KubeClient.CoreV1().Services(namespace)
	return serviceClient.Get(name, metaV1.GetOptions{})
}

// ValidServicePort returns valid Service object if given service with the port exists in the given namespace
func ValidServicePort(name, namespace string, port int32) (*apiV1.Service, error) {
	serviceClient := utils.KubeClient.CoreV1().Services(namespace)
	service, err := serviceClient.Get(name, metaV1.GetOptions{})
	if err != nil {
		return service, err
	}
	for _, p := range service.Spec.Ports {
		if p.Port == port {
			return service, nil
		}
	}
	return service, fmt.Errorf("Port %d is not exposed by the service %s", port, name)
}

// ValidSecret return Secret object if the secret is present in the specified object
func ValidSecret(name, namespace string) (*apiV1.Secret, error) {
	secretClient := utils.KubeClient.CoreV1().Secrets(namespace)
	return secretClient.Get(name, metaV1.GetOptions{})
}

// FindNamespaceFromService returns namespace from fully qualified domain name
func FindNamespaceFromService(service string) string {
	ns := strings.Split(service, ".")
	if len(ns) > 1 {
		return ns[1]
	}
	return "default"
}
