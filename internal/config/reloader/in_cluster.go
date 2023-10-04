package reloader

import (
	"context"
	"fmt"
	"reflect"
	"time"

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
	labelKey                    = "botkube.io/config-watch"
	labelValue                  = "true"
	defaultInformerResyncPeriod = 60 * time.Second
	dataKey                     = "data"
)

var (
	selector         = labels.SelectorFromSet(map[string]string{labelKey: labelValue}).String()
	tweakListOptions = func(options *metav1.ListOptions) {
		options.LabelSelector = selector
	}

	configMapGVR = schema.GroupVersionResource{Version: metav1.SchemeGroupVersion.Version, Group: "", Resource: "configmaps"}
	secretsGVR   = schema.GroupVersionResource{Version: metav1.SchemeGroupVersion.Version, Group: "", Resource: "secrets"}
)

var _ Reloader = &InClusterConfigReloader{}

type InClusterConfigReloader struct {
	log       logrus.FieldLogger
	cli       dynamic.Interface
	cfg       config.CfgWatcher
	reporter  analytics.Reporter
	restarter *Restarter

	informerFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewInClusterConfigReloader(log logrus.FieldLogger, cli dynamic.Interface, cfg config.CfgWatcher, restarter *Restarter, reporter analytics.Reporter) (*InClusterConfigReloader, error) {
	informerResyncPeriod := cfg.InCluster.InformerResyncPeriod
	if informerResyncPeriod == 0 {
		informerResyncPeriod = defaultInformerResyncPeriod
	}
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

	_, err = l.informerFactory.ForResource(secretsGVR).Informer().AddEventHandler(eventHandler)
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

var _ cache.ResourceEventHandler = &genericEventHandler{}

type genericEventHandler struct {
	log       logrus.FieldLogger
	ctx       context.Context
	restarter *Restarter
}

func newGenericEventHandler(ctx context.Context, log logrus.FieldLogger, restarter *Restarter) *genericEventHandler {
	return &genericEventHandler{log: log, ctx: ctx, restarter: restarter}
}

func (g *genericEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	g.log.WithFields(logrus.Fields{
		"obj":             formatx.StructDumper().Sdump(obj),
		"isInInitialList": isInInitialList,
	}).Debug("OnCreate called")

	if isInInitialList {
		g.log.Debug("OnAdd: initial list. Skipping...")
		return
	}
	g.log.Infoln("OnAdd", formatx.StructDumper().Sdump(obj), isInInitialList)
	g.reloadIfCan(g.ctx)
}

// OnUpdate is called when an object is updated.
//
// We don't need to handle the case when new object has removed the label. That case is handled by OnDelete.
func (g *genericEventHandler) OnUpdate(oldObj, newObj interface{}) {
	g.log.WithFields(logrus.Fields{
		"old": formatx.StructDumper().Sdump(oldObj),
		"new": formatx.StructDumper().Sdump(newObj),
	}).Debug("OnUpdate called")

	unstrOldObj, ok := oldObj.(*unstructured.Unstructured)
	if !ok {
		g.log.Errorf("unexpected type of old object: %T", oldObj)
		return
	}

	unstrNewObj, ok := newObj.(*unstructured.Unstructured)
	if !ok {
		g.log.Errorf("unexpected type of new object: %T", newObj)
		return
	}

	if unstrOldObj.GetResourceVersion() == unstrNewObj.GetResourceVersion() {
		g.log.Debug("OnUpdate: resource version is the same. Skipping...")
		return
	}

	g.log.Debug("Comparing content...")
	// both Secret and ConfigMap have Data field

	oldData := unstrOldObj.Object[dataKey]
	newData := unstrNewObj.Object[dataKey]

	if reflect.DeepEqual(oldData, newData) {
		g.log.Debug("OnUpdate: content is the same. Skipping...")
		return
	}

	g.reloadIfCan(g.ctx)
}

func (g *genericEventHandler) OnDelete(obj interface{}) {
	g.log.Infoln("OnDelete", formatx.StructDumper().Sdump(obj))
	g.reloadIfCan(g.ctx)
}

func (g *genericEventHandler) reloadIfCan(ctx context.Context) {
	g.log.Info("Reloading configuration...")
	err := g.restarter.Do(ctx)
	if err != nil {
		g.log.Errorf("while restarting the app: %s", err.Error())
		return
	}
}
