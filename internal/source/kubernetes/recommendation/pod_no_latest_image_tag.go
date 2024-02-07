package recommendation

import (
	"context"
	"fmt"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/k8sutil"
	"github.com/kubeshop/botkube/pkg/k8sx"
)

const podNoLatestImageTag = "PodNoLatestImageTag"

// PodNoLatestImageTag add recommendations if latest image tag is used in Pod containers.
type PodNoLatestImageTag struct{}

// NewPodNoLatestImageTag creates a new PodNoLatestImageTag instance.
func NewPodNoLatestImageTag() *PodNoLatestImageTag {
	return &PodNoLatestImageTag{}
}

// Do executes the recommendation checks.
func (f *PodNoLatestImageTag) Do(_ context.Context, event event.Event) (Result, error) {
	if event.Kind != "Pod" || event.Type != config.CreateEvent || k8sutil.GetObjectTypeMetaData(event.Object).Kind == "Event" {
		return Result{}, nil
	}

	unstrObj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return Result{}, fmt.Errorf("cannot convert %T into type %T", event.Object, unstrObj)
	}

	var pod coreV1.Pod
	err := k8sx.TransformIntoTypedObject(unstrObj, &pod)
	if err != nil {
		return Result{}, fmt.Errorf("while transforming object type %T into type: %T: %w", event.Object, pod, err)
	}

	podIdentifier := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

	infoMsgs := f.checkContainers("initContainer", pod.Spec.InitContainers, podIdentifier)
	infoMsgs = append(infoMsgs, f.checkContainers("container", pod.Spec.Containers, podIdentifier)...)

	return Result{
		Info: infoMsgs,
	}, nil
}

func (f *PodNoLatestImageTag) checkContainers(fieldName string, containers []coreV1.Container, podIdentifier string) []string {
	var recomms []string
	for _, c := range containers {
		images := strings.Split(c.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			recommendationMsg := fmt.Sprintf("The 'latest' tag used in '%s' image of Pod '%s' %s '%s' should be avoided.", c.Image, podIdentifier, fieldName, c.Name)
			recomms = append(recomms, recommendationMsg)
		}
	}

	return recomms
}

// Name returns the recommendation name.
func (f *PodNoLatestImageTag) Name() string {
	return podNoLatestImageTag
}
