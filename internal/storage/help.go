package storage

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HelpEntries defines the help persistence model.
type HelpEntries map[string]bool

const helpKey = "help-message"

// Help provides functionality to persist the information about sent help messages.
type Help struct {
	systemConfigMapName      string
	systemConfigMapNamespace string

	k8sCli kubernetes.Interface
}

// NewForHelp returns a new Help instance.
func NewForHelp(ns, name string, k8sCli kubernetes.Interface) *Help {
	return &Help{
		systemConfigMapNamespace: ns,
		systemConfigMapName:      name,
		k8sCli:                   k8sCli,
	}
}

// GetSentHelpDetails returns details about sent help messages.
func (a *Help) GetSentHelpDetails(ctx context.Context) (HelpEntries, error) {
	obj, err := a.k8sCli.CoreV1().ConfigMaps(a.systemConfigMapNamespace).Get(ctx, a.systemConfigMapName, metav1.GetOptions{})
	switch {
	case err == nil:
	case apierrors.IsNotFound(err):
		return HelpEntries{}, nil
	default:
		return HelpEntries{}, fmt.Errorf("while getting the Config Map: %w", err)
	}

	return a.extractHelpDetails(obj)
}

// MarkHelpAsSent marks a given sent keys as sent.
func (a *Help) MarkHelpAsSent(ctx context.Context, sent []string) error {
	alreadySent := HelpEntries{}
	for _, item := range sent {
		alreadySent[item] = true
	}
	rawSent, err := json.Marshal(alreadySent)
	if err != nil {
		return fmt.Errorf("while marshaling input keys: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.systemConfigMapName,
			Namespace: a.systemConfigMapNamespace,
		},
		Data: map[string]string{
			helpKey: string(rawSent),
		},
	}

	_, err = a.k8sCli.CoreV1().ConfigMaps(a.systemConfigMapNamespace).Create(ctx, cm, metav1.CreateOptions{})
	switch {
	case err == nil:
	case apierrors.IsAlreadyExists(err):
		old, err := a.k8sCli.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("while getting already existing ConfigMap: %w", err)
		}

		previousEntries, err := a.extractHelpDetails(old)
		if err != nil {
			return fmt.Errorf("while extracting help details: %w", err)
		}

		for _, item := range sent {
			previousEntries[item] = true
		}

		newRawSent, err := json.Marshal(previousEntries)
		if err != nil {
			return fmt.Errorf("while marshaling final output: %w", err)
		}

		newCM := old.DeepCopy()
		if newCM.Data == nil {
			newCM.Data = map[string]string{}
		}
		newCM.Data[helpKey] = string(newRawSent)

		_, err = a.k8sCli.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, newCM, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("while updating the ConfigMap with help details: %w", err)
		}

	default:
		return fmt.Errorf("while creating the ConfigMap with help details: %w", err)
	}

	return nil
}

func (a *Help) extractHelpDetails(cm *corev1.ConfigMap) (HelpEntries, error) {
	data, found := cm.Data[helpKey]
	if !found {
		return HelpEntries{}, nil
	}

	out := HelpEntries{}
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return HelpEntries{}, fmt.Errorf("while unmarhshaling the help data: %w", err)
	}
	return out, nil
}
