package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/kubeshop/botkube/pkg/api/source"
)

const pluginName = "cm-watcher"

type (
	// Config holds executor configuration.
	Config struct {
		ConfigMap Object
	}
	// Object holds information about object to watch.
	Object struct {
		Name      string          `yaml:"name"`
		Namespace string          `yaml:"namespace"`
		Event     watch.EventType `yaml:"event"`
	}
)

// CMWatcher implements Botkube source plugin.
type CMWatcher struct{}

// Stream returns a given command as response.
func (CMWatcher) Stream(ctx context.Context) (source.StreamOutput, error) {
	// TODO: in request we should receive the executor configuration.
	cfg := Config{
		ConfigMap: Object{
			Name:      "cm-watcher-trigger",
			Namespace: "botkube",
			Event:     "ADDED",
		},
	}

	out := source.StreamOutput{
		Output: make(chan []byte),
	}

	go listenEvents(ctx, cfg.ConfigMap, out.Output)

	return out, nil
}

func listenEvents(ctx context.Context, obj Object, sink chan<- []byte) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	exitOnError(err)
	clientset, err := kubernetes.NewForConfig(config)
	exitOnError(err)

	fieldSelector := fields.OneTermEqualSelector("metadata.name", obj.Name).String()
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return clientset.CoreV1().ConfigMaps(obj.Namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return clientset.CoreV1().ConfigMaps(obj.Namespace).Watch(ctx, options)
		},
	}

	infiniteWatch := func(event watch.Event) (bool, error) {
		if event.Type == obj.Event {
			cm := event.Object.(*corev1.ConfigMap)
			msg := fmt.Sprintf("Detected `%s` event on `%s/%s`", obj.Event, cm.Namespace, cm.Name)
			sink <- []byte(msg)
		}

		// always continue - context will cancel this watch for us :)
		return false, nil
	}

	_, err = watchtools.UntilWithSync(ctx, lw, &corev1.ConfigMap{}, nil, infiniteWatch)
	exitOnError(err)
}

func main() {
	source.Serve(map[string]plugin.Plugin{
		pluginName: &source.Plugin{
			Source: &CMWatcher{},
		},
	})
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
