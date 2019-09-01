package filters

import (
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/utils"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DisableAnnotation is the object disable annotation
	DisableAnnotation string = "botkube.io/disable"
	// ChannelAnnotation is the multichannel support annotation
	ChannelAnnotation string = "botkube.io/channel"
)

// ObjectAnnotationChecker add recommendations to the event object if pod created without any labels
type ObjectAnnotationChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(ObjectAnnotationChecker{
		Description: "Checks if annotations botkube.io/* present in object specs and filters them.",
	})
}

// Run filters and modifies event struct
func (f ObjectAnnotationChecker) Run(object interface{}, event *events.Event) {

	// get objects metadata
	obj := utils.GetObjectMetaData(object)

	// Check annotations in object
	if isObjectNotifDisabled(obj) {
		event.Skip = true
		log.Logger.Debug("Object Notification Disable through annotations")
	}

	if channel, ok := reconfigureChannel(obj); ok {
		event.Channel = channel
		log.Logger.Debugf("Redirecting Event Notifications to channel: %s", channel)
	}

	log.Logger.Debug("Object annotations filter successful!")
}

// Describe filter
func (f ObjectAnnotationChecker) Describe() string {
	return f.Description
}

// isObjectNotifDisabled checks annotation botkube.io/disable
// annotation botkube.io/disable disables the event notifications from objects
func isObjectNotifDisabled(obj metaV1.ObjectMeta) bool {

	if obj.Annotations[DisableAnnotation] == "true" {
		log.Logger.Debug("Skipping Disabled Event Notifications!")
		return true
	}
	return false
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
