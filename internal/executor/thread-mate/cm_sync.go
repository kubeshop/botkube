package thread_mate

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// dataFieldName is the key used to store data in the ConfigMap.
const dataFieldName = "data"

// ConfigMapDumper is a utility for working with Kubernetes ConfigMaps.
type ConfigMapDumper struct {
	k8sCli kubernetes.Interface
}

// NewConfigMapDumper creates a new instance of ConfigMapDumper.
func NewConfigMapDumper(k8sCli kubernetes.Interface) *ConfigMapDumper {
	return &ConfigMapDumper{
		k8sCli: k8sCli,
	}
}

// SaveOrUpdate saves or updates data in a ConfigMap in the specified namespace.
func (a *ConfigMapDumper) SaveOrUpdate(namespace, name, data string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			dataFieldName: data,
		},
	}

	ctx := context.Background()
	_, err := a.k8sCli.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	switch {
	case err == nil:
	case apierrors.IsAlreadyExists(err):
		old, err := a.k8sCli.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("while getting already existing ConfigMap: %w", err)
		}

		newCM := old.DeepCopy()
		if newCM.Data == nil {
			newCM.Data = map[string]string{}
		}
		newCM.Data[dataFieldName] = data

		_, err = a.k8sCli.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, newCM, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("while updating ConfigMap: %w", err)
		}

	default:
		return fmt.Errorf("while creating ConfigMap: %w", err)
	}
	return nil
}

// Get retrieves data from a ConfigMap in the specified namespace.
func (a *ConfigMapDumper) Get(namespace, name string) (string, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	ctx := context.Background()
	cm, err := a.k8sCli.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("while getting ConfigMap: %w", err)
	}

	return cm.Data[dataFieldName], nil
}
