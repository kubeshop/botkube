package reloader

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/formatx"
)

const (
	labelKey   = "botkube.io/config-watch"
	labelValue = "true"
	dataKey    = "data"
)

var (
	labelSelector    = labels.SelectorFromSet(map[string]string{labelKey: labelValue}).String()
	tweakListOptions = func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	configMapGVR = schema.GroupVersionResource{Version: metav1.SchemeGroupVersion.Version, Group: "", Resource: "configmaps"}
	secretGVR    = schema.GroupVersionResource{Version: metav1.SchemeGroupVersion.Version, Group: "", Resource: "secrets"}
)

var _ Reloader = &InClusterConfigReloader{}

type restarter interface {
	Do(ctx context.Context) error
}

type InClusterConfigReloader struct {
	log       logrus.FieldLogger
	cli       dynamic.Interface
	cfg       config.CfgWatcher
	reporter  analytics.Reporter
	restarter restarter

	informerFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewInClusterConfigReloader(log logrus.FieldLogger, cli dynamic.Interface, cfg config.CfgWatcher, restarter restarter, reporter analytics.Reporter) (*InClusterConfigReloader, error) {
	informerResyncPeriod := cfg.InCluster.InformerResyncPeriod
	informerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(cli, informerResyncPeriod, cfg.Deployment.Namespace, tweakListOptions)
	return &InClusterConfigReloader{log: log, cli: cli, cfg: cfg, reporter: reporter, restarter: restarter, informerFactory: informerFactory}, nil
}

func (l *InClusterConfigReloader) Do(ctx context.Context) error {
	l.log.Info("Adding event handlers...")
	eventHandler := newGenericEventHandler(ctx, l.log.WithField("subcomponent", "genericEventHandler"), l.restarter)

	_, err := l.informerFactory.ForResource(configMapGVR).Informer().AddEventHandler(eventHandler)
	if err != nil {
		return fmt.Errorf("while adding event handler for configmaps: %w", err)
	}
	_, err = l.informerFactory.ForResource(secretGVR).Informer().AddEventHandler(eventHandler)
	if err != nil {
		return fmt.Errorf("while adding event handler for secrets: %w", err)
	}

	l.log.Info("Starting informers...")
	l.informerFactory.Start(ctx.Done())
	l.log.Info("Waiting for cache sync...")
	l.informerFactory.WaitForCacheSync(ctx.Done())
	l.log.Info("Cache synced successfully.")
	defer l.informerFactory.Shutdown()

	<-ctx.Done()
	l.log.Info("Exiting...")
	return nil
}

func (l *InClusterConfigReloader) InformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return l.informerFactory
}

var _ cache.ResourceEventHandler = &genericEventHandler{}

type genericEventHandler struct {
	log       logrus.FieldLogger
	ctx       context.Context
	restarter restarter
}

func newGenericEventHandler(ctx context.Context, log logrus.FieldLogger, restarter restarter) *genericEventHandler {
	return &genericEventHandler{log: log, ctx: ctx, restarter: restarter}
}

func (g *genericEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	log := g.log.WithField("type", "OnAdd")
	log.WithFields(logrus.Fields{
		"obj":             formatx.StructDumper().Sdump(obj),
		"isInInitialList": isInInitialList,
	}).Debug("Handling event...")

	if isInInitialList {
		log.Debug("this is the initial list. Skipping reloading...")
		return
	}

	unstrObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("unexpected type of object: %T", obj)
		return
	}

	// This shouldn't happen at all, as we use FilteredDynamicSharedInformerFactory
	// which filters out objects without the label. However, fake K8s client doesn't seem to support labelSelector in tweakListOptions - at least in OnAdd events.
	// Controversial decision here: I decided to handle this case anyway, just in case.
	if unstrObj.GetLabels()[labelKey] != labelValue {
		log.Debug("label is not set. Skipping...")
		return
	}

	g.reloadIfCan(g.ctx)
}

// OnUpdate is called when an object is updated.
//
// We don't need to handle the case when new object has removed the label. That case is handled by OnDelete.
func (g *genericEventHandler) OnUpdate(oldObj, newObj interface{}) {
	log := g.log.WithField("type", "OnUpdate")
	log.WithFields(logrus.Fields{
		"old": formatx.StructDumper().Sdump(oldObj),
		"new": formatx.StructDumper().Sdump(newObj),
	}).Debug("Handling event...")

	unstrOldObj, ok := oldObj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Unexpected type of old object: %T", oldObj)
		return
	}

	unstrNewObj, ok := newObj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Unexpected type of new object: %T", newObj)
		return
	}

	if unstrOldObj.GetResourceVersion() == unstrNewObj.GetResourceVersion() {
		log.Debug("Resource version is the same. Skipping...")
		return
	}

	log.Debug("Comparing content...")
	// both Secret and ConfigMap have Data field
	oldData := unstrOldObj.Object[dataKey]
	newData := unstrNewObj.Object[dataKey]

	if reflect.DeepEqual(oldData, newData) {
		g.log.Debug("Content is the same. Skipping...")
		return
	}

	g.reloadIfCan(g.ctx)
}

func (g *genericEventHandler) OnDelete(obj interface{}) {
	g.log.WithFields(logrus.Fields{
		"obj":  formatx.StructDumper().Sdump(obj),
		"type": "OnDelete",
	}).Debug("Handling event...")
	g.reloadIfCan(g.ctx)
}

func (g *genericEventHandler) reloadIfCan(ctx context.Context) {
	g.log.Debug("Reloading configuration...")
	err := g.restarter.Do(ctx)
	if err != nil {
		g.log.Errorf("while restarting the app: %s", err.Error())
		return
	}
}
