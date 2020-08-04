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

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/utils"
)

// PodLabelChecker add recommendations to the event object if pod created without any labels
type PodLabelChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(PodLabelChecker{
		Description: "Checks and adds recommedations if labels are missing in the pod specs.",
	})
}

// Run filters and modifies event struct
func (f PodLabelChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" || event.Type != config.CreateEvent {
		return
	}

	podObjectMeta := utils.GetObjectMetaData(object)

	// Check labels in pod
	if len(podObjectMeta.Labels) == 0 {
		event.Recommendations = append(event.Recommendations, fmt.Sprintf("pod '%s' creation without labels should be avoided.", podObjectMeta.Name))
	}
	log.Debug("Pod label filter successful!")
}

// Describe filter
func (f PodLabelChecker) Describe() string {
	return f.Description
}
