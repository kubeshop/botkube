package k8sutil

import (
	"context"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type TypeMeta struct {
	Kind       string
	APIVersion string
}

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

		eventAnnotations, err := extractAnnotationsFromEvent(ctx, dynamicCli, mapper, &eventObj)
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
func GetObjectTypeMetaData(obj interface{}) TypeMeta {
	k, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return TypeMeta{}
	}
	return TypeMeta{
		APIVersion: k.GetAPIVersion(),
		Kind:       k.GetKind(),
	}
}

// GetResourceFromKind returns resource name for given Kind
func GetResourceFromKind(mapper meta.RESTMapper, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("Error while creating REST Mapping for Event Involved Object: %v", err)
	}
	return mapping.Resource, nil
}

// TransformIntoTypedObject uses unstructured interface and creates a typed object
func TransformIntoTypedObject(obj *unstructured.Unstructured, typedObject interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObject)
}

// extractAnnotationsFromEvent returns annotations of a related resource for the given event.
func extractAnnotationsFromEvent(ctx context.Context, dynamicCli dynamic.Interface, mapper meta.RESTMapper, obj *coreV1.Event) (map[string]string, error) {
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
