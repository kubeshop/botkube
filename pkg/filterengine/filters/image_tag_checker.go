package filters

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/utils"
)

// ImageTagChecker add recommendations to the event object if latest image tag is used in pod containers
type ImageTagChecker struct {
	log logrus.FieldLogger
}

// NewImageTagChecker creates a new ImageTagChecker instance
func NewImageTagChecker(log logrus.FieldLogger) *ImageTagChecker {
	return &ImageTagChecker{log: log}
}

// Run filers and modifies event struct
func (f *ImageTagChecker) Run(_ context.Context, object interface{}, event *events.Event) error {
	if event.Kind != "Pod" || event.Type != config.CreateEvent || utils.GetObjectTypeMetaData(object).Kind == "Event" {
		return nil
	}
	var podObj coreV1.Pod
	err := utils.TransformIntoTypedObject(object.(*unstructured.Unstructured), &podObj)
	if err != nil {
		return fmt.Errorf("while transforming object type %T into type: %T: %w", object, podObj, err)
	}

	// Check image tag in initContainers
	for _, ic := range podObj.Spec.InitContainers {
		images := strings.Split(ic.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, fmt.Sprintf(":latest tag used in image '%s' of initContainer '%s' should be avoided.", ic.Image, ic.Name))
		}
	}

	// Check image tag in Containers
	for _, c := range podObj.Spec.Containers {
		images := strings.Split(c.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, fmt.Sprintf(":latest tag used in image '%s' of Container '%s' should be avoided.", c.Image, c.Name))
		}
	}
	f.log.Debug("Image tag filter successful!")
	return nil
}

// Name returns the filter's name
func (f *ImageTagChecker) Name() string {
	return "ImageTagChecker"
}

// Describe describes the filter
func (f *ImageTagChecker) Describe() string {
	return "Checks and adds recommendation if 'latest' image tag is used for container image."
}
