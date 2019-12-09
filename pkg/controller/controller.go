package controller

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"

	// Register filters
	_ "github.com/infracloudio/botkube/pkg/filterengine/filters"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerStartMsg = "...and now my watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "my watch has ended for cluster '%s'!"
	configUpdateMsg    = "Looks like the configuration is updated for cluster '%s'. I shall halt my watch till I read it."
)

var startTime time.Time

// RegisterInformers creates new informer controllers to watch k8s resources
func RegisterInformers(c *config.Config, notifiers []notify.Notifier) {
	sendMessage(c, notifiers, fmt.Sprintf(controllerStartMsg, c.Settings.ClusterName))
	startTime = time.Now()

	// Start config file watcher if enabled
	if c.Settings.ConfigWatcher {
		go configWatcher(c, notifiers)
	}

	// Register informers for resource lifecycle events
	if len(c.Resources) > 0 {
		log.Logger.Info("Registering resource lifecycle informer")
		for _, r := range c.Resources {
			if _, ok := utils.ResourceInformerMap[r.Name]; !ok {
				continue
			}
			log.Logger.Infof("Adding informer for resource:%s", r.Name)
			utils.ResourceInformerMap[r.Name].AddEventHandler(registerEventHandlers(c, notifiers, r.Name, r.Events))
		}
	}

	// Register informers for k8s events
	log.Logger.Infof("Registering kubernetes events informer for types: %+v", config.WarningEvent.String())
	log.Logger.Infof("Registering kubernetes events informer for types: %+v", config.NormalEvent.String())

	utils.KubeInformerFactory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			_, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Logger.Errorf("Failed to get MetaNamespaceKey from event resource")
				return
			}
			eventObj, ok := obj.(*coreV1.Event)
			if !ok {
				return
			}

			// Kind of involved object
			kind := strings.ToLower(eventObj.InvolvedObject.Kind)

			switch strings.ToLower(eventObj.Type) {
			case config.WarningEvent.String():
				// Send WarningEvent as ErrorEvents
				sendEvent(obj, nil, c, notifiers, kind, config.ErrorEvent)
			case config.NormalEvent.String():
				// Send NormalEvent as Insignificant InfoEvent
				sendEvent(obj, nil, c, notifiers, kind, config.InfoEvent)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	utils.KubeInformerFactory.Start(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
	sendMessage(c, notifiers, fmt.Sprintf(controllerStopMsg, c.Settings.ClusterName))
}

func registerEventHandlers(c *config.Config, notifiers []notify.Notifier, resourceType string, events []config.EventType) (handlerFns cache.ResourceEventHandlerFuncs) {
	for _, event := range events {
		if event == config.AllEvent || event == config.CreateEvent {
			handlerFns.AddFunc = func(obj interface{}) {
				log.Logger.Debugf("Processing add to %v", resourceType)
				sendEvent(obj, nil, c, notifiers, resourceType, config.CreateEvent)
			}
		}

		if event == config.AllEvent || event == config.UpdateEvent {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				log.Logger.Debugf("Processing update to %v\n Object: %+v\n", resourceType, new)
				sendEvent(new, old, c, notifiers, resourceType, config.UpdateEvent)
			}
		}

		if event == config.AllEvent || event == config.DeleteEvent {
			handlerFns.DeleteFunc = func(obj interface{}) {
				log.Logger.Debugf("Processing delete to %v", resourceType)
				sendEvent(obj, nil, c, notifiers, resourceType, config.DeleteEvent)
			}
		}
	}
	return handlerFns
}

func sendEvent(obj, oldObj interface{}, c *config.Config, notifiers []notify.Notifier, kind string, eventType config.EventType) {
	// Filter namespaces
	objectMeta := utils.GetObjectMetaData(obj)

	switch eventType {
	case config.InfoEvent:
		// Skip if ErrorEvent is not configured for the resource
		if !utils.AllowedEventKindsMap[utils.EventKind{Resource: kind, Namespace: "all", EventType: config.ErrorEvent}] &&
			!utils.AllowedEventKindsMap[utils.EventKind{Resource: kind, Namespace: objectMeta.Namespace, EventType: config.ErrorEvent}] {
			log.Logger.Debugf("Ignoring %s to %s/%v in %s namespaces", eventType, kind, objectMeta.Name, objectMeta.Namespace)
			return
		}
	default:
		if !utils.AllowedEventKindsMap[utils.EventKind{Resource: kind, Namespace: "all", EventType: eventType}] &&
			!utils.AllowedEventKindsMap[utils.EventKind{Resource: kind, Namespace: objectMeta.Namespace, EventType: eventType}] {
			log.Logger.Debugf("Ignoring %s to %s/%v in %s namespaces", eventType, kind, objectMeta.Name, objectMeta.Namespace)
			return
		}
	}

	log.Logger.Debugf("Processing %s to %s/%v in %s namespaces", eventType, kind, objectMeta.Name, objectMeta.Namespace)

	// Check if Notify disabled
	if !config.Notify {
		log.Logger.Debug("Skipping notification")
		return
	}

	// Create new event object
	event := events.New(obj, eventType, kind, c.Settings.ClusterName)

	// Skip older events
	if !event.TimeStamp.IsZero() {
		if event.TimeStamp.Before(startTime) {
			log.Logger.Debug("Skipping older events")
			return
		}
	}

	// check for siginificant Update Events in objects
	if eventType == config.UpdateEvent {
		updateMsg := utils.Diff(oldObj, obj)
		if len(updateMsg) > 0 {
			event.Messages = append(event.Messages, updateMsg)
		} else {
			// skipping least significant update
			log.Logger.Debug("skipping least significant Update event")
			event.Skip = true
		}
	}

	// Filter events
	event = filterengine.DefaultFilterEngine.Run(obj, event)
	if event.Skip {
		log.Logger.Debugf("Skipping event: %#v", event)
		return
	}

	// Skip unpromoted insignificant InfoEvents
	if event.Type == config.InfoEvent {
		log.Logger.Debugf("Skipping Insignificant InfoEvent: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		log.Logger.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	// check if Recommendations are disabled
	if !c.Recommendations {
		event.Recommendations = nil
		log.Logger.Debug("Skipping Recommendations in Event Notifications")
	}

	// Send event over notifiers
	for _, n := range notifiers {
		go n.SendEvent(event)
	}
}

func sendMessage(c *config.Config, notifiers []notify.Notifier, msg string) {
	if len(msg) <= 0 {
		log.Logger.Warn("sendMessage received string with length 0. Hence skipping.")
		return
	}

	// Send message over notifiers
	for _, n := range notifiers {
		go n.SendMessage(msg)
	}
}

func configWatcher(c *config.Config, notifiers []notify.Notifier) {
	configPath := os.Getenv("CONFIG_PATH")
	configFile := filepath.Join(configPath, config.ResourceConfigFileName)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Logger.Fatal("Failed to create file watcher ", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					log.Logger.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
					return
				}
				log.Logger.Infof("Config file %s is updated. Hence restarting the Pod", configFile)
				done <- true

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Logger.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
					return
				}
			}
		}
	}()
	log.Logger.Infof("Registering watcher on configfile %s", configFile)
	err = watcher.Add(configFile)
	if err != nil {
		log.Logger.Errorf("Unable to register watch on config file:%s. Error: %s", configFile, err.Error())
		return
	}
	<-done
	sendMessage(c, notifiers, fmt.Sprintf(configUpdateMsg, c.Settings.ClusterName))
	// Wait for Notifier to send message
	time.Sleep(5 * time.Second)
	os.Exit(0)
}
