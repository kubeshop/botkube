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
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"

	coreV1 "k8s.io/api/core/v1"
)

// ImageTagChecker add recommendations to the event object if latest image tag is used in pod containers
type ImageTagChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(ImageTagChecker{
		Description: "Checks and adds recommendation if 'latest' image tag is used for container image.",
	})
}

// Run filers and modifies event struct
func (f ImageTagChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" || event.Type != config.CreateEvent {
		return
	}
	podObj, ok := object.(*coreV1.Pod)
	if !ok {
		return
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
	log.Debug("Image tag filter successful!")
}

// Describe filter
func (f ImageTagChecker) Describe() string {
	return f.Description
}
