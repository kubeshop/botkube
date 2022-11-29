package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

// Config holds executor configuration.
type (
	Config struct {
		ConfigMap Object
	}
	Object struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
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
		},
	}

	out := source.StreamOutput{
		Output: make(chan []byte),
	}

	go listenEvents(ctx, cfg.ConfigMap, out.Output)

	return out, nil
}

func listenEvents(ctx context.Context, obj Object, sink chan<- []byte) {
	home, err := os.UserHomeDir()
	exitOnError(err)

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	exitOnError(err)
	clientset, err := kubernetes.NewForConfig(config)
	exitOnError(err)

	fieldSelector := fields.OneTermEqualSelector("metadata.name", obj.Name).String()
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return clientset.CoreV1().Pods(obj.Namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return clientset.CoreV1().Pods(obj.Namespace).Watch(ctx, options)
		},
	}

	fmt.Println("starting informer")
	_, informer, watcher, _ := watchtools.NewIndexerInformerWatcher(lw, &corev1.ConfigMap{})
	defer watcher.Stop()

	fmt.Println("waiting for cache sync")
	cache.WaitForCacheSync(ctx.Done(), informer.HasSynced)

	ch := watcher.ResultChan()
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-ch:
			fmt.Println("get event", event)
			if !ok { // finished
				return
			}
			cm := event.Object.(*corev1.ConfigMap)
			sink <- []byte(cm.Name)
		case <-ctx.Done(): // client closed streaming
			return
		}
	}
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
