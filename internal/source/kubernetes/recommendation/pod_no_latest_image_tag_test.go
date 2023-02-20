package recommendation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
)

func TestPodNoLatestImageTag_Do_HappyPath(t *testing.T) {
	// given
	expected := recommendation.Result{
		Info: []string{
			"The 'latest' tag used in 'foo' image of Pod 'foo/pod-name' initContainer 'first-init' should be avoided.",
			"The 'latest' tag used in 'bar:latest' image of Pod 'foo/pod-name' initContainer 'second-init' should be avoided.",
			"The 'latest' tag used in 'foo' image of Pod 'foo/pod-name' container 'first' should be avoided.",
			"The 'latest' tag used in 'bar:latest' image of Pod 'foo/pod-name' container 'second' should be avoided.",
		},
	}

	recomm := recommendation.NewPodNoLatestImageTag()

	pod := fixPod()
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pod)
	require.NoError(t, err)
	unstr := &unstructured.Unstructured{Object: unstrObj}

	event, err := event.New(pod.ObjectMeta, unstr, config.CreateEvent, "v1/pods")
	require.NoError(t, err)

	// when
	actual, err := recomm.Do(context.Background(), event)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func fixPod() *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-name",
			Namespace: "foo",
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{Name: "first-init", Image: "foo"},
				{Name: "second-init", Image: "bar:latest"},
				{Name: "third-init", Image: "baz:v1"},
				{Name: "fourth-init", Image: "qux:3.1.4"},
			},
			Containers: []v1.Container{
				{Name: "first", Image: "foo"},
				{Name: "second", Image: "bar:latest"},
				{Name: "third", Image: "baz:v1"},
				{Name: "fourth", Image: "qux:3.1.4"},
			},
		},
	}
}
