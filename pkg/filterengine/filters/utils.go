// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package filters

import (
	"context"
	"fmt"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

// ValidService returns Service object is service given service exists in the given namespace
func ValidService(ctx context.Context, name, namespace string) (*coreV1.Service, error) {
	unstructuredService, err := utils.DynamicKubeClient.Resource(serviceGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var serviceObject coreV1.Service
	err = utils.TransformIntoTypedObject(unstructuredService, &serviceObject)
	if err != nil {
		return nil, err
	}
	return &serviceObject, nil
}

// ValidServicePort returns valid Service object if given service with the port exists in the given namespace
func ValidServicePort(ctx context.Context, name, namespace string, port int32) (*coreV1.Service, error) {
	unstructuredService, err := utils.DynamicKubeClient.Resource(serviceGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
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
func ValidSecret(ctx context.Context, name, namespace string) (*coreV1.Secret, error) {
	unstructuredSecret, err := utils.DynamicKubeClient.Resource(secretGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
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
