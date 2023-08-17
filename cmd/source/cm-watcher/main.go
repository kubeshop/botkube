package main

import (
	"context"
	"fmt"
	"log"

	"github.com/MakeNowJust/heredoc"
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

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

const (
	pluginName  = "cm-watcher"
	description = "Kubernetes ConfigMap watcher is an example Botkube source plugin used during e2e tests. It's not meant for production usage."
)

type (
	// Config holds executor configuration.
	Config struct {
		ConfigMap Object `yaml:"configMap,omitempty"`
	}
	// Object holds information about object to watch.
	Object struct {
		Name      string          `yaml:"name,omitempty"`
		Namespace string          `yaml:"namespace,omitempty"`
		Event     watch.EventType `yaml:"event,omitempty"`
	}
)

var defaultConfig = Config{
	ConfigMap: Object{
		Name:      "cm-watcher-trigger",
		Namespace: "default",
		Event:     "ADDED",
	},
}

// CMWatcher implements Botkube source plugin.
type CMWatcher struct{}

// Metadata returns details about ConfigMap watcher plugin.
func (CMWatcher) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     version,
		Description: description,
		JSONSchema:  jsonSchema(),
	}, nil
}

// Stream sends an event when a given ConfigMap is matched against the criteria defined in config.
func (CMWatcher) Stream(ctx context.Context, in source.StreamInput) (source.StreamOutput, error) {
	var cfg Config
	err := pluginx.MergeSourceConfigsWithDefaults(defaultConfig, in.Configs, &cfg)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configuration: %w", err)
	}

	out := source.StreamOutput{
		Event: make(chan source.Event),
	}

	go listenEvents(ctx, in.Context.KubeConfig, cfg.ConfigMap, out.Event)

	return out, nil
}

func listenEvents(ctx context.Context, kubeConfig []byte, obj Object, sink chan source.Event) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
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
			msg := fmt.Sprintf("Plugin %s detected `%s` event on `%s/%s`", pluginName, obj.Event, cm.Namespace, cm.Name)
			sink <- source.Event{
				Message: api.NewPlaintextMessage(msg, true),
			}
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

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "botkube/cm-watcher",
			"description": "%s",
			"type": "object",
			"properties": {},
			"required": []
		}`, description),
	}
}
