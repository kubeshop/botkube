package controller

import (
	//"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/infracloudio/kubeops/pkg/config"
	"github.com/infracloudio/kubeops/pkg/events"
	"github.com/infracloudio/kubeops/pkg/filterengine"
	"github.com/infracloudio/kubeops/pkg/logging"
	"github.com/infracloudio/kubeops/pkg/notify"
	"github.com/infracloudio/kubeops/pkg/utils"

	apiV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func findNamespace(ns string) string {
	if ns == "all" {
		return apiV1.NamespaceAll
	}
	if ns == "" {
		return apiV1.NamespaceDefault
	}
	return ns
}

func RegisterInformers(c *config.Config) {
	// Register informers for resource lifecycle events
	if len(c.Resources) > 0 {
		logging.Logger.Info("Registering resource lifecycle informer")
		for _, r := range c.Resources {
			if _, ok := utils.ResourceGetterMap[r.Name]; !ok {
				continue
			}
			object, ok := utils.RtObjectMap[r.Name]
			if !ok {
				continue
			}
			for _, ns := range r.Namespaces {
				logging.Logger.Infof("Adding informer for resource:%s namespace:%s", r.Name, ns)

				watchlist := cache.NewListWatchFromClient(
					utils.ResourceGetterMap[r.Name], r.Name, findNamespace(ns), fields.Everything())

				_, controller := cache.NewInformer(
					watchlist,
					object,
					0*time.Second,
					registerEventHandlers(r.Name, r.Events),
				)
				stopCh := make(chan struct{})
				defer close(stopCh)

				go controller.Run(stopCh)

			}
		}
	}

	// Register informers for k8s events
	if len(c.Events.Types) > 0 {
		logging.Logger.Info("Registering kubernetes events informer")
		watchlist := cache.NewListWatchFromClient(
			utils.KubeClient.CoreV1().RESTClient(), "events", apiV1.NamespaceAll, fields.Everything())

		_, controller := cache.NewInformer(
			watchlist,
			&apiV1.Event{},
			0*time.Second,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					eventObj, ok := obj.(*apiV1.Event)
					if !ok {
						return
					}

					kind := strings.ToLower(eventObj.InvolvedObject.Kind)
					ns := eventObj.InvolvedObject.Namespace
					eType := strings.ToLower(eventObj.Type)

					logging.Logger.Debugf("Received event:- kind:%s ns:%s type:%s", kind, ns, eType)
					// Filter and forward
					if (utils.AllowedEventKindsMap[utils.EventKind{kind, "all"}] ||
						utils.AllowedEventKindsMap[utils.EventKind{kind, ns}]) && (utils.AllowedEventTypesMap[eType]) {
						logging.Logger.Infof("Processing add to events: %s. Invoked Object: %s:%s", key, eventObj.InvolvedObject.Kind, eventObj.InvolvedObject.Namespace)
						logEvent(obj, "events", "create", err)
					}
				},
			},
		)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go controller.Run(stopCh)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm

}

func registerEventHandlers(resourceType string, events []string) (handlerFns cache.ResourceEventHandlerFuncs) {
	for _, event := range events {
		if event == "all" || event == "create" {
			handlerFns.AddFunc = func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				logging.Logger.Infof("Processing add to %v: %s", resourceType, key)
				logEvent(obj, resourceType, "create", err)
			}
		}

		if event == "all" || event == "update" {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				logging.Logger.Infof("Processing update to %v: %s", resourceType, key)
				logEvent(new, resourceType, "update", err)
			}
		}

		if event == "all" || event == "delete" {
			handlerFns.DeleteFunc = func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				logging.Logger.Infof("Processing delete to %v: %s", resourceType, key)
				logEvent(obj, resourceType, "delete", err)
			}
		}
	}
	return handlerFns
}

func logEvent(obj interface{}, kind, eventType string, err error) error {
	if err != nil {
		logging.Logger.Error("Error while receiving event: ", err.Error())
	}
	if !config.Notify {
		logging.Logger.Info("Skipping notification")
		return nil
	}
	event := events.New(obj, eventType, kind)
	event = filterengine.DefaultFilterEngine.Run(obj, event)

	notifier := notify.NewSlack()
	notifier.Send(event)

	return nil
}
