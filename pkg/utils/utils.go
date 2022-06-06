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

package utils

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const hyperlinkRegex = `(?m)<http:\/\/[a-z.0-9\/\-_=]*\|([a-z.0-9\/\-_=]*)>`

// GetObjectMetaData returns metadata of the given object
func GetObjectMetaData(ctx context.Context, dynamicCli dynamic.Interface, mapper meta.RESTMapper, obj interface{}) (metaV1.ObjectMeta, error) {
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return metaV1.ObjectMeta{}, fmt.Errorf("cannot convert type %T into *unstructured.Unstructured", obj)
	}
	unstructuredObject = unstructuredObject.DeepCopy()
	objectMeta := metaV1.ObjectMeta{
		Name:                       unstructuredObject.GetName(),
		GenerateName:               unstructuredObject.GetGenerateName(),
		Namespace:                  unstructuredObject.GetNamespace(),
		ResourceVersion:            unstructuredObject.GetResourceVersion(),
		Generation:                 unstructuredObject.GetGeneration(),
		CreationTimestamp:          unstructuredObject.GetCreationTimestamp(),
		DeletionTimestamp:          unstructuredObject.GetDeletionTimestamp(),
		DeletionGracePeriodSeconds: unstructuredObject.GetDeletionGracePeriodSeconds(),
		Labels:                     unstructuredObject.GetLabels(),
		Annotations:                unstructuredObject.GetAnnotations(),
		OwnerReferences:            unstructuredObject.GetOwnerReferences(),
		Finalizers:                 unstructuredObject.GetFinalizers(),
		ManagedFields:              unstructuredObject.GetManagedFields(),
	}
	if GetObjectTypeMetaData(obj).Kind == "Event" {
		var eventObj coreV1.Event
		err := TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
		if err != nil {
			return metaV1.ObjectMeta{}, fmt.Errorf("while transforming object type: %T into type %T: %w", obj, eventObj, err)
		}

		eventAnnotations, err := ExtractAnnotationsFromEvent(ctx, dynamicCli, mapper, &eventObj)
		if err != nil {
			return metaV1.ObjectMeta{}, err
		}

		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		for key, value := range eventAnnotations {
			objectMeta.Annotations[key] = value
		}
	}
	return objectMeta, nil
}

// GetObjectTypeMetaData returns typemetadata of the given object
func GetObjectTypeMetaData(obj interface{}) metaV1.TypeMeta {
	k, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return metaV1.TypeMeta{}
	}
	k = k.DeepCopy()
	return metaV1.TypeMeta{
		APIVersion: k.GetAPIVersion(),
		Kind:       k.GetKind(),
	}
}

// DeleteDoubleWhiteSpace returns slice that removing whitespace from a arg slice
func DeleteDoubleWhiteSpace(slice []string) []string {
	var result []string
	for _, s := range slice {
		if len(s) != 0 {
			result = append(result, s)
		}
	}
	return result
}

// GetResourceFromKind returns resource name for given Kind
func GetResourceFromKind(mapper meta.RESTMapper, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("Error while creating REST Mapping for Event Involved Object: %v", err)
	}
	return mapping.Resource, nil
}

// ExtractAnnotationsFromEvent returns annotations of InvolvedObject for the given event
func ExtractAnnotationsFromEvent(ctx context.Context, dynamicCli dynamic.Interface, mapper meta.RESTMapper, obj *coreV1.Event) (map[string]string, error) {
	gvr, err := GetResourceFromKind(mapper, obj.InvolvedObject.GroupVersionKind())
	if err != nil {
		return nil, err
	}
	annotations, err := dynamicCli.Resource(gvr).Namespace(obj.InvolvedObject.Namespace).Get(ctx, obj.InvolvedObject.Name, metaV1.GetOptions{})
	if err != nil {
		// IgnoreNotFound returns nil on NotFound errors.
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return annotations.GetAnnotations(), nil
}

//GetClusterNameFromKubectlCmd this will return cluster name from kubectl command
func GetClusterNameFromKubectlCmd(cmd string) string {
	r, _ := regexp.Compile(`--cluster-name[=|' ']([^\s]*)`)
	//this gives 2 match with cluster name and without
	matchedArray := r.FindStringSubmatch(cmd)
	var s string
	if len(matchedArray) >= 2 {
		s = matchedArray[1]
	}
	return s
}

// GVRToString converts GVR formats to string
func GVRToString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}

// TransformIntoTypedObject uses unstructured interface and creates a typed object
func TransformIntoTypedObject(obj *unstructured.Unstructured, typedObject interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObject)
}

//GetStringInYamlFormat get the formatted commands list
func GetStringInYamlFormat(header string, commands map[string]bool) string {
	var b bytes.Buffer
	fmt.Fprintln(&b, header)
	for k, v := range commands {
		if v {
			fmt.Fprintf(&b, "  - %s\n", k)
		}
	}
	return b.String()
}

// Contains tells whether a contains x.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if strings.EqualFold(x, n) {
			return true
		}
	}
	return false
}

// RemoveHyperlink removes the hyperlink text from url
func RemoveHyperlink(hyperlink string) string {
	command := hyperlink
	compiledRegex := regexp.MustCompile(hyperlinkRegex)
	matched := compiledRegex.FindAllStringSubmatch(string(hyperlink), -1)
	if len(matched) >= 1 {
		for _, match := range matched {
			if len(match) == 2 {
				command = strings.ReplaceAll(command, match[0], match[1])
			}
		}
	}
	return command
}
