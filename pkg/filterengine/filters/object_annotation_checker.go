package filters

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/utils"
)

const (
	// DisableAnnotation is the object disable annotation
	DisableAnnotation string = "botkube.io/disable"
	// ChannelAnnotation is the multichannel support annotation
	ChannelAnnotation string = "botkube.io/channel"
)

// ObjectAnnotationChecker add recommendations to the event object if pod created without any labels
type ObjectAnnotationChecker struct {
	log        logrus.FieldLogger
	dynamicCli dynamic.Interface
	mapper     meta.RESTMapper
}

// NewObjectAnnotationChecker creates a new ObjectAnnotationChecker instance
func NewObjectAnnotationChecker(log logrus.FieldLogger, dynamicCli dynamic.Interface, mapper meta.RESTMapper) *ObjectAnnotationChecker {
	return &ObjectAnnotationChecker{log: log, dynamicCli: dynamicCli, mapper: mapper}
}

// Run filters and modifies event struct
func (f *ObjectAnnotationChecker) Run(ctx context.Context, object interface{}, event *events.Event) error {
	// get objects metadata
	obj, err := utils.GetObjectMetaData(ctx, f.dynamicCli, f.mapper, object)
	if err != nil {
		return fmt.Errorf("while getting object metadata: %w", err)
	}

	// Check annotations in object
	if f.isObjectNotifDisabled(obj) {
		event.Skip = true
		f.log.Debug("Object Notification Disable through annotations")
	}

	if channel, ok := f.reconfigureChannel(obj); ok {
		event.Channel = channel
		f.log.Debugf("Redirecting Event Notifications to channel: %s", channel)
	}

	f.log.Debug("Object annotations filter successful!")
	return nil
}

// Name returns the filter's name
func (f *ObjectAnnotationChecker) Name() string {
	return "ObjectAnnotationChecker"
}

// Describe describes the filter
func (f *ObjectAnnotationChecker) Describe() string {
	return "Checks if annotations botkube.io/* present in object specs and filters them."
}

// isObjectNotifDisabled checks annotation botkube.io/disable
// annotation botkube.io/disable disables the event notifications from objects
func (f *ObjectAnnotationChecker) isObjectNotifDisabled(obj metaV1.ObjectMeta) bool {
	if obj.Annotations[DisableAnnotation] == "true" {
		f.log.Debug("Skipping Disabled Event Notifications!")
		return true
	}
	return false
}

// reconfigureChannel checks annotation botkube.io/channel
// annotation botkube.io/channel directs event notifications to channels
// based on the channel names present in them
// Note: Add botkube app into the desired channel to receive notifications
func (f *ObjectAnnotationChecker) reconfigureChannel(obj metaV1.ObjectMeta) (string, bool) {
	// redirect messages to channels based on annotations
	if channel, ok := obj.Annotations[ChannelAnnotation]; ok {
		return channel, true
	}
	return "", false
}
