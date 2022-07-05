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
