package kubex

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

// WaitForPod watches a given Pod until the exitCondition is true.
func WaitForPod(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string, exitCondition watchtools.ConditionFunc) error {
	selector := labels.SelectorFromSet(map[string]string{"app": name}).String()
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = selector
			return clientset.CoreV1().Pods(namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = selector
			return clientset.CoreV1().Pods(namespace).Watch(ctx, options)
		},
	}

	_, err := watchtools.UntilWithSync(ctx, lw, &corev1.Pod{}, nil, exitCondition)
	return err
}

var (
	errPodRestartedWithError = errors.New("pod restarted with non zero exit code")
)

// PodReady returns true if the Pod is read.
func PodReady(podScheduledIndicator chan string, since time.Time) func(event watch.Event) (bool, error) {
	informed := false
	sinceK8sTime := metav1.NewTime(since)
	return func(event watch.Event) (bool, error) {
		switch t := event.Type; t {
		case watch.Added, watch.Modified:
			switch pod := event.Object.(type) {
			case *corev1.Pod:

				createdAt := pod.GetObjectMeta().GetCreationTimestamp()
				// we don't care about previously created pods, for example when user do some upgrades, we watch for a new Pod instance only.
				if createdAt.Before(&sinceK8sTime) {
					return false, nil
				}

				if pod.Status.Phase == corev1.PodRunning && !informed {
					informed = true
					podScheduledIndicator <- pod.Name
					close(podScheduledIndicator)
				}

				for _, cond := range pod.Status.ContainerStatuses {
					if !cond.Ready && cond.RestartCount > 0 {
						// pod was already restarted because of the problem, we restart botkube on permanent errors mostly, so let's stop watching
						return true, errPodRestartedWithError
					}
				}

				return isPodReady(pod), nil
			}
		}

		return false, nil
	}
}

// isPodReady returns true if a pod is ready; false otherwise.
func isPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
