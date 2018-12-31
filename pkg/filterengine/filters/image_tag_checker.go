package filters

import (
	"strings"

	"github.com/infracloudio/kubeops/pkg/events"
	log "github.com/infracloudio/kubeops/pkg/logging"

	apiV1 "k8s.io/api/core/v1"
)

// ImageTagChecker add recommendations to the event object if latest image tag is used in pod containers
type ImageTagChecker struct {
}

// NewImageTagChecker creates new ImageTagChecker object
func NewImageTagChecker() *ImageTagChecker {
	return &ImageTagChecker{}
}

// Run filers and modifies event struct
func (f *ImageTagChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" && event.Type != "create" {
		return
	}
	podObj, ok := object.(*apiV1.Pod)
	if !ok {
		return
	}

	// Check image tag in initContainers
	for _, ic := range podObj.Spec.InitContainers {
		images := strings.Split(ic.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, ":latest tag used in image '"+ic.Image+"' of initContainer '"+ic.Name+"' should be avoided.\n")
		}
	}

	// Check image tag in Containers
	for _, c := range podObj.Spec.Containers {
		images := strings.Split(c.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, ":latest tag used in image '"+c.Image+"' of Container '"+c.Name+"' should be avoided.\n")
		}
	}
	log.Logger.Info("Image tag filter successful!")
}
