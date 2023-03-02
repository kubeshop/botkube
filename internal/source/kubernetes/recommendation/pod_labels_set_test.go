package recommendation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
)

func TestPodLabelsSet_Do_HappyPath(t *testing.T) {
	// given
	expected := recommendation.Result{
		Info: []string{
			"Pod 'foo/pod-name' created without labels. Consider defining them, to be able to use them as a selector e.g. in Service.",
		},
	}

	recomm := recommendation.NewPodLabelsSet()

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
