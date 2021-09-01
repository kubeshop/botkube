// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package filters

import (
	"context"
	"strconv"

	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/patrickmn/go-cache"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	// DisableAnnotation is the object disable annotation
	DisableAnnotation string = "botkube.io/disable"
	// ChannelAnnotation is the multichannel support annotation
	ChannelAnnotation string = "botkube.io/channel"
	// NotificationAnnotation is the namespace level notification disable annotation
	NotificationAnnotation string = "botkube.io/notification"
)

// ObjectAnnotationChecker add recommendations to the event object if pod created without any labels
type ObjectAnnotationChecker struct {
	Description string
}

var (
	// Cache for storing NotificationAnnotation info
	NamespaceCache *cache.Cache
)

// Register filter
func init() {

	// Create a cache with no expiration
	NamespaceCache = cache.New(cache.NoExpiration, cache.NoExpiration)

	filterengine.DefaultFilterEngine.Register(ObjectAnnotationChecker{
		Description: "Checks if annotations botkube.io/* present in object specs and filters them.",
	})
}

// Run filters and modifies event struct
func (f ObjectAnnotationChecker) Run(object interface{}, event *events.Event) {

	// get objects metadata
	obj := utils.GetObjectMetaData(object)

	// Check NotificationAnnotation on object's namespace
	if !isObjectNamespaceNotifEnabled(obj) {
		event.Skip = true
		log.Debugf("Object notification disabled through namespace's '%s' annotation", NotificationAnnotation)
	}

	// Check annotations in object
	if isObjectNotifDisabled(obj) {
		event.Skip = true
		log.Debug("Object Notification Disable through annotations")
	}

	if channel, ok := reconfigureChannel(obj); ok {
		event.Channel = channel
		log.Debugf("Redirecting Event Notifications to channel: %s", channel)
	}

	log.Debug("Object annotations filter successful!")
}

// Describe filter
func (f ObjectAnnotationChecker) Describe() string {
	return f.Description
}

// isObjectNotifDisabled checks annotation botkube.io/disable
// annotation botkube.io/disable disables the event notifications from objects
func isObjectNotifDisabled(obj metaV1.ObjectMeta) bool {

	if obj.Annotations[DisableAnnotation] == "true" {
		log.Debug("Skipping Disabled Event Notifications!")
		return true
	}
	return false
}

// isObjectNamespaceNotifEnabled checks annotation botkube.io/notification on the namespace
// annotation botkube.io/notification enable/disable all object's event notification in the namespace
func isObjectNamespaceNotifEnabled(obj metaV1.ObjectMeta) bool {

	namespace := obj.Namespace
	nsObjectNotifEnabled, found := NamespaceCache.Get(namespace)

	if !found {
		log.Warnf("Cache Miss!! Namespace '%s' not found in the cache!", namespace)

		// get namespace object
		nsObject, err := utils.DynamicKubeClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Get(context.TODO(), namespace, v1.GetOptions{})

		if err != nil {
			log.Error(err)
			nsObjectNotifEnabled = "false"
		} else {
			annotationsList := nsObject.GetAnnotations()
			val := annotationsList[NotificationAnnotation]
			if val != "false" {
				val = "true"
			}

			NamespaceCache.Set(nsObject.GetName(), val, cache.NoExpiration)
			nsObjectNotifEnabled = val
			log.Debugf("Namespace Cache: %v", NamespaceCache.Items())
		}

	} else {
		log.Infof("Cache Hit!! Namespace '%s' found in the cache!", namespace)
	}

	b1, _ := strconv.ParseBool(nsObjectNotifEnabled.(string))
	return b1

}

// Registers watch for namespace's events
func RegisterWatchNamespace() error {

	log.Info("Registering namespace watch")

	// register watch on namespaces
	nsWatcher, err := utils.DynamicKubeClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Watch(context.TODO(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watchNamespaceEvent(nsWatcher.ResultChan())

	return nil
}

// Watches for namespace's events via channel
// Updates the respective namespace's botkube.io/notification info in the NamespaceCache
func watchNamespaceEvent(nsEventChan <-chan watch.Event) {

	for event := range nsEventChan {

		notifiedObj := event.Object.(*unstructured.Unstructured)
		namespace := notifiedObj.GetName()

		log.Debug("Received event in watch namespace")
		log.Debugf("Namespace: %s", namespace)
		log.Debugf("Annotations: %s", notifiedObj.GetAnnotations())

		annotationsList := notifiedObj.GetAnnotations()
		val := annotationsList[NotificationAnnotation]

		log.Debugf("Checking if namespace '%s' exists", namespace)

		// check existence of namespace
		_, err := utils.DynamicKubeClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Get(context.TODO(), namespace, v1.GetOptions{})
		if err != nil {
			// namespace does not exist
			log.Debugf("Namespace '%s' does not exist!!", namespace)
			log.Error(err)
			// delete namespace entry from cache
			NamespaceCache.Delete(notifiedObj.GetName())
			log.Debugf("Namespace Cache: %v", NamespaceCache.Items())

		} else {
			// namespace exists
			log.Debugf("Namespace '%s' exists!", namespace)
			if val != "false" {
				val = "true"
			}
			NamespaceCache.Set(notifiedObj.GetName(), val, cache.NoExpiration)
			log.Debugf("Namespace Cache: %v", NamespaceCache.Items())

		}
	}

}

// reconfigureChannel checks annotation botkube.io/channel
// annotation botkube.io/channel directs event notifications to channels
// based on the channel names present in them
// Note: Add botkube app into the desired channel to receive notifications
func reconfigureChannel(obj metaV1.ObjectMeta) (string, bool) {
	// redirect messages to channels based on annotations
	if channel, ok := obj.Annotations[ChannelAnnotation]; ok {
		return channel, true
	}
	return "", false
}
