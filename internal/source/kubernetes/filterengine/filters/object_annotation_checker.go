package filters

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/k8sutil"
)

const (
	// DisableAnnotation is the object disable annotation.
	DisableAnnotation string = "botkube.io/disable"
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
