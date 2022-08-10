package recommendation

import (
	"context"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/utils"
)

const podLabelsSetName = "PodLabelsSet"

// PodLabelsSet adds recommendation when newly created Pods have no labels.
type PodLabelsSet struct{}

// NewPodLabelsSet creates a new PodLabelsSet instance.
func NewPodLabelsSet() *PodLabelsSet {
	return &PodLabelsSet{}
}

// Do executes the recommendation checks.
func (f PodLabelsSet) Do(_ context.Context, event events.Event) (Result, error) {
	if event.Kind != "Pod" || event.Type != config.CreateEvent {
		return Result{}, nil
	}

	//podObjectMeta, err := utils.GetObjectMetaData(ctx, f.dynamicCli, f.mapper, event.Object)
	//if err != nil {
	//	return fmt.Errorf("while getting object metadata: %w", err)
	//}

	unstrObj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return Result{}, fmt.Errorf("cannot convert %T into type %T", event.Object, unstrObj)
	}

	var pod coreV1.Pod
	err := utils.TransformIntoTypedObject(unstrObj, &pod)
	if err != nil {
		return Result{}, fmt.Errorf("while transforming object type %T into type: %T: %w", event.Object, pod, err)
	}

	if len(pod.Labels) > 0 {
		return Result{}, nil
	}

	recommendationMsg := fmt.Sprintf("Pod '%s/%s' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.", pod.Namespace, pod.Name)
	return Result{
		Info: []string{recommendationMsg},
	}, nil
}

// Name returns the recommendation name.
func (f *PodLabelsSet) Name() string {
	return podLabelsSetName
}
