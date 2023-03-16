package insights_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fake2 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	testing2 "k8s.io/client-go/testing"

	"github.com/kubeshop/botkube/internal/insights"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/status"
)

func Test_Start_Success(t *testing.T) {
	wrkNode1 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker1",
		},
	}
	wrkNode2 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker2",
		},
	}

	k8sCli := fake.NewSimpleClientset(&wrkNode1, &wrkNode2)
	statusReporter := status.NoopStatusReporter{}

	collector := insights.NewK8sCollector(k8sCli, statusReporter, loggerx.NewNoop(), 1, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()
	err := collector.Start(ctx)
	assert.True(t, retry.IsRecoverable(err))
}

func Test_Start_Failed(t *testing.T) {
	k8sCli := fake.NewSimpleClientset()
	k8sCli.CoreV1().(*fake2.FakeCoreV1).PrependReactor("list", "nodes", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.NodeList{}, errors.New("error listing nodes")
	})
	statusReporter := status.NoopStatusReporter{}

	collector := insights.NewK8sCollector(k8sCli, statusReporter, loggerx.NewNoop(), 1, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()
	err := collector.Start(ctx)
	assert.Contains(t, err.Error(), "reached maximum limit of node count retrieval")
}
