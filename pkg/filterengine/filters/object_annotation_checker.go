package filters

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/k8sutil"
)

const (
	// DisableAnnotation is the object disable annotation.
	DisableAnnotation string = "botkube.io/disable"
	// ChannelAnnotation is the multichannel support annotation.
	ChannelAnnotation string = "botkube.io/channel"
)

// ObjectAnnotationChecker forwards events to specific channels based on a special annotation if it is set on a given K8s resource.
type ObjectAnnotationChecker struct {
	log        logrus.FieldLogger
	dynamicCli dynamic.Interface
	mapper     meta.RESTMapper
}

// NewObjectAnnotationChecker creates a new ObjectAnnotationChecker instance.
func NewObjectAnnotationChecker(log logrus.FieldLogger, dynamicCli dynamic.Interface, mapper meta.RESTMapper) *ObjectAnnotationChecker {
	return &ObjectAnnotationChecker{log: log, dynamicCli: dynamicCli, mapper: mapper}
}

// Run filters and modifies event struct.
func (f *ObjectAnnotationChecker) Run(ctx context.Context, event *event.Event) error {
	// get objects metadata
	obj, err := k8sutil.GetObjectMetaData(ctx, f.dynamicCli, f.mapper, event.Object)
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

// Name returns the filter's name.
func (f *ObjectAnnotationChecker) Name() string {
	return "ObjectAnnotationChecker"
}

// Describe describes the filter.
func (f *ObjectAnnotationChecker) Describe() string {
	return "Filters or reroutes events based on botkube.io/* Kubernetes resource annotations."
}

// isObjectNotifDisabled checks annotation botkube.io/disable.
// Annotation botkube.io/disable disables the event notifications from objects.
func (f *ObjectAnnotationChecker) isObjectNotifDisabled(obj metaV1.ObjectMeta) bool {
	if obj.Annotations[DisableAnnotation] == "true" {
		f.log.Debug("Skipping Disabled Event Notifications!")
		return true
	}
	return false
}

// reconfigureChannel checks annotation botkube.io/channel.
// Annotation botkube.io/channel directs event notifications to channels
// based on the channel names present in them.
// Note: Add botkube app into the desired channel to receive notifications
func (f *ObjectAnnotationChecker) reconfigureChannel(obj metaV1.ObjectMeta) (string, bool) {
	// redirect messages to channels based on annotations
	if channel, ok := obj.Annotations[ChannelAnnotation]; ok {
		return channel, true
	}
	return "", false
}
