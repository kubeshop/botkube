package reloader_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/botkube/internal/config/reloader"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestRestarter_Do_HappyPath(t *testing.T) {
	// given
	clusterName := "foo"

	expectedMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", clusterName)

	deployCfg := config.K8sResourceRef{
		Name:      "name",
		Namespace: "namespace",
	}
	inputDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployCfg.Name,
			Namespace: deployCfg.Namespace,
		},
	}
	sendMsgFn := reloader.SendMessageFn(func(msg string) error {
		assert.Equal(t, expectedMsg, msg)
		return nil
	})

	k8sCli := fake.NewSimpleClientset(inputDeploy)

	restarter := reloader.NewRestarter(loggerx.NewNoop(), k8sCli, deployCfg, clusterName, sendMsgFn)

	// when
	err := restarter.Do(context.Background())

	// then
	require.NoError(t, err)

	actualDeploy, err := k8sCli.AppsV1().Deployments(deployCfg.Namespace).Get(context.Background(), deployCfg.Name, metav1.GetOptions{})
	require.NoError(t, err)

	_, exists := actualDeploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	assert.True(t, exists)
}
