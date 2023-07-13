package helm

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

// waitForTestSuite watches the given test suite until the exitCondition is true
func WaitForBotkubePod(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string, exitCondition watchtools.ConditionFunc, timeout time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	//preconditionFunc := func(store cache.Store) (bool, error) {
	//	_, exists, err := store.Get(&metav1.ObjectMeta{Name: name, Namespace: namespace})
	//	if err != nil {
	//		fmt.Println(err)
	//		return true, err
	//	}
	//	if !exists {
	//		// We need to make sure we see the object in the cache before we start waiting for events
	//		// or we would be waiting for the timeout if such object didn't exist.
	//		fmt.Println("exists", exists)
	//		return true, apierrors.NewNotFound(corev1.Resource("pods"), name)
	//	}
	//
	//	return false, nil
	//}

	//selector := labels.SelectorFromSet(map[string]string{"app": name}).String()
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			//options.LabelSelector = selector
			return clientset.CoreV1().Pods(namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			//options.LabelSelector = selector
			return clientset.CoreV1().Pods(namespace).Watch(ctx, options)
		},
	}

	_, err := watchtools.UntilWithSync(ctx, lw, &corev1.Pod{}, nil, exitCondition)
	return err
}

var (
	errPodRestartedWithError = errors.New("pod restarted with non zero exit code")
)

// clusterTestSuiteCompleted returns true if the suite has run to completion, false if the suite has not yet
// reached running state, or an error in any other case.
func PodReady(podScheduledIndicator, phase chan string, since time.Time) func(event watch.Event) (bool, error) {
	//infiniteWatch := func(event watch.Event) (bool, error) {
	//	if event.Type == obj.Event {
	//		cm := event.Object.(*corev1.ConfigMap)
	//		msg := fmt.Sprintf("Plugin %s detected `%s` event on `%s/%s`", pluginName, obj.Event, cm.Namespace, cm.Name)
	//		sink <- []byte(msg)
	//	}
	//
	//	// always continue - context will cancel this watch for us :)
	//	return false, nil
	//}

	informed := false

	sinceK8sTime := metav1.NewTime(since)
	//prevPodName := "" // when we do upgrade we will just ignore it
	return func(event watch.Event) (bool, error) {
		//fmt.Printf("testing %v\n", event)
		switch t := event.Type; t {
		case watch.Added, watch.Modified:
			switch pod := event.Object.(type) {
			case *corev1.Pod:

				createdAt := pod.GetObjectMeta().GetCreationTimestamp()
				if createdAt.Before(&sinceK8sTime) {
					return false, nil
				}

				phase <- string(pod.Status.Phase)
				if pod.Status.Phase == corev1.PodRunning && !informed {
					informed = true
					podScheduledIndicator <- pod.Name
					close(podScheduledIndicator)
				}

				for _, cond := range pod.Status.ContainerStatuses {
					if cond.Ready == false && cond.RestartCount > 0 {
						// pod was already restarted because of the problem, we restart botkube on permanent errors mostly, so let's stop watching
						return true, errPodRestartedWithError
					}
				}

				//fmt.Printf("Pod phase: %q\n", pod.Status.Phase)
				return isPodReady(pod), nil
			}
			//case watch.Deleted:
			// We need to abort to avoid cases of recreation and not to silently watch the wrong (new) object
			//return false, apierrors.NewNotFound(corev1.Resource("pods"), "")
			//default:
			//	fmt.Println("test")
			//	return true, fmt.Errorf("internal error: unexpected event %#v", event)
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
		//fmt.Printf("Pod is  ready: %s/%s\n", pod.GetNamespace(), pod.GetName())
		//fmt.Println("Finishing!!")
	}
	//fmt.Printf("Pod is not ready: %s/%s\n", pod.GetNamespace(), pod.GetName())
	return false
}
