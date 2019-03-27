package controller

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"

	apiV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerStartMsg = "...and now my watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "my watch has ended for cluster '%s'!"
)

var startTime time.Time

func findNamespace(ns string) string {
	if ns == "all" {
		return apiV1.NamespaceAll
	}
	if ns == "" {
		return apiV1.NamespaceDefault
	}
	return ns
}

// RegisterInformers creates new informer controllers to watch k8s resources
func RegisterInformers(c *config.Config) {
	sendMessage(c, fmt.Sprintf(controllerStartMsg, c.Settings.ClusterName))
	startTime = time.Now().Local()

	// Get resync period
	rsyncTimeStr, ok := os.LookupEnv("INFORMERS_RESYNC_PERIOD")
	if !ok {
		rsyncTimeStr = "30"
	}

	rsyncTime, err := strconv.Atoi(rsyncTimeStr)
	if err != nil {
		log.Logger.Fatal("Error in reading INFORMERS_RESYNC_PERIOD env var.", err)
	}

	// Register informers for resource lifecycle events
	if len(c.Resources) > 0 {
		log.Logger.Info("Registering resource lifecycle informer")
		for _, r := range c.Resources {
			if _, ok := utils.ResourceGetterMap[r.Name]; !ok {
				continue
			}
			object, ok := utils.RtObjectMap[r.Name]
			if !ok {
				continue
			}
			for _, ns := range r.Namespaces {
				log.Logger.Infof("Adding informer for resource:%s namespace:%s", r.Name, ns)

				watchlist := cache.NewListWatchFromClient(
					utils.ResourceGetterMap[r.Name], r.Name, findNamespace(ns), fields.Everything())

				_, controller := cache.NewInformer(
					watchlist,
					object,
					time.Duration(rsyncTime)*time.Minute,
					registerEventHandlers(c, r.Name, r.Events),
				)
				stopCh := make(chan struct{})
				defer close(stopCh)

				go controller.Run(stopCh)

			}
		}
	}

	// Register informers for k8s events
	if len(c.Events.Types) > 0 {
		log.Logger.Info("Registering kubernetes events informer")
		watchlist := cache.NewListWatchFromClient(
			utils.KubeClient.CoreV1().RESTClient(), "events", apiV1.NamespaceAll, fields.Everything())

		_, controller := cache.NewInformer(
			watchlist,
			&apiV1.Event{},
			time.Duration(rsyncTime)*time.Minute,
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

					log.Logger.Debugf("Received event: kind:%s ns:%s type:%s", kind, ns, eType)
					// Filter and forward
					if (utils.AllowedEventKindsMap[utils.EventKind{kind, "all"}] ||
						utils.AllowedEventKindsMap[utils.EventKind{kind, ns}]) && (utils.AllowedEventTypesMap[eType]) {
						log.Logger.Infof("Processing add to events: %s. Invoked Object: %s:%s", key, eventObj.InvolvedObject.Kind, eventObj.InvolvedObject.Namespace)
						sendEvent(obj, c, "events", "create", err)
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
	sendMessage(c, fmt.Sprintf(controllerStopMsg, c.Settings.ClusterName))
}

func registerEventHandlers(c *config.Config, resourceType string, events []string) (handlerFns cache.ResourceEventHandlerFuncs) {
	for _, event := range events {
		if event == "all" || event == "create" {
			handlerFns.AddFunc = func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				log.Logger.Debugf("Processing add to %v: %s", resourceType, key)
				sendEvent(obj, c, resourceType, "create", err)
			}
		}

		if event == "all" || event == "update" {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				log.Logger.Debugf("Processing update to %v: %s", resourceType, key)
				sendEvent(new, c, resourceType, "update", err)
			}
		}

		if event == "all" || event == "delete" {
			handlerFns.DeleteFunc = func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				log.Logger.Debugf("Processing delete to %v: %s", resourceType, key)
				sendEvent(obj, c, resourceType, "delete", err)
			}
		}
	}
	return handlerFns
}

func sendEvent(obj interface{}, c *config.Config, kind, eventType string, err error) {
	if err != nil {
		log.Logger.Error("Error while receiving event: ", err.Error())
		return
	}

	// Skip older events
	if eventType == "create" {
		objectMeta := utils.GetObjectMetaData(obj)
		if objectMeta.CreationTimestamp.Sub(startTime).Seconds() <= 0 {
			log.Logger.Debug("Skipping older events")
			return
		}
	}

	// Skip older events
	if eventType == "delete" {
		objectMeta := utils.GetObjectMetaData(obj)
		if objectMeta.DeletionTimestamp != nil && objectMeta.DeletionTimestamp.Sub(startTime).Seconds() <= 0 {
			log.Logger.Debug("Skipping older events")
			return
		}
	}

	// Check if Notify disabled
	if !config.Notify {
		log.Logger.Debug("Skipping notification")
		return
	}

	// Create new event object
	event := events.New(obj, eventType, kind)
	event = filterengine.DefaultFilterEngine.Run(obj, event)
	if event.Skip {
		log.Logger.Debugf("Skipping event: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		log.Logger.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	var notifier notify.Notifier
	// Send notification to communication channel
	if c.Communications.Slack.Enable {
		notifier = notify.NewSlack(c)
		go notifier.SendEvent(event)
	}

	if c.Communications.ElasticSearch.Enable {
		notifier = notify.NewElasticSearch(c)
		go notifier.SendEvent(event)
	}
}

func sendMessage(c *config.Config, msg string) {
	if len(msg) <= 0 {
		log.Logger.Warn("sendMessage received string with length 0. Hence skipping.")
		return
	}
	if c.Communications.Slack.Enable {
		notifier := notify.NewSlack(c)
		go notifier.SendMessage(msg)
	}
}
