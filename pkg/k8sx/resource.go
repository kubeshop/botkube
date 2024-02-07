package k8sx

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// TransformIntoTypedObject uses unstructured interface and creates a typed object.
func TransformIntoTypedObject(obj *unstructured.Unstructured, typedObject interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObject)
}

// TransformIntoUnstructured uses typed object and creates an unstructured interface.
func TransformIntoUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: out}, nil
}
