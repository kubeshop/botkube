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
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"

	"github.com/infracloudio/botkube/pkg/utils"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
