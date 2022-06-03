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

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/utils"
)

// PodLabelChecker add recommendations to the event object if pod created without any labels
type PodLabelChecker struct {
	log        logrus.FieldLogger
	dynamicCli dynamic.Interface
	mapper     meta.RESTMapper
}

// NewPodLabelChecker creates a new PodLabelChecker instance
func NewPodLabelChecker(log logrus.FieldLogger, dynamicCli dynamic.Interface, mapper meta.RESTMapper) *PodLabelChecker {
	return &PodLabelChecker{log: log, dynamicCli: dynamicCli, mapper: mapper}
}

// Run filters and modifies event struct
func (f PodLabelChecker) Run(ctx context.Context, object interface{}, event *events.Event) error {
	if event.Kind != "Pod" || event.Type != config.CreateEvent {
		return nil
	}

	podObjectMeta, err := utils.GetObjectMetaData(ctx, f.dynamicCli, f.mapper, object)
	if err != nil {
		return fmt.Errorf("while getting object metadata: %w", err)
	}

	// Check labels in pod
	if len(podObjectMeta.Labels) == 0 {
		event.Recommendations = append(event.Recommendations, fmt.Sprintf("pod '%s' creation without labels should be avoided.", podObjectMeta.Name))
	}
	f.log.Debug("Pod label filter successful!")
	return nil
}

// Name returns the filter's name
func (f *PodLabelChecker) Name() string {
	return "PodLabelChecker"
}

// Describe describes the filter
func (f PodLabelChecker) Describe() string {
	return "Checks and adds recommendations if labels are missing in the pod specs."
}
