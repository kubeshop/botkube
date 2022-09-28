package lifecycle

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestNewReloadHandler_HappyPath(t *testing.T) {
	// given
	clusterName := "foo"

	expectedMsg := fmt.Sprintf(":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec...", clusterName)
	expectedResponse := `Deployment "namespace/name" restarted successfully.`

	expectedStatusCode := http.StatusOK
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
	sendMsgFn := SendMessageFn(func(msg string) error {
		assert.Equal(t, expectedMsg, msg)
		return nil
	})
	logger, _ := logtest.NewNullLogger()
	k8sCli := fake.NewSimpleClientset(inputDeploy)

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	writer := httptest.NewRecorder()
	handler := newReloadHandler(logger, k8sCli, deployCfg, clusterName, sendMsgFn)

	// when
	handler(writer, req)

	res := writer.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	// then
	assert.Equal(t, expectedStatusCode, res.StatusCode)
	assert.Equal(t, expectedResponse, string(data))

	actualDeploy, err := k8sCli.AppsV1().Deployments(deployCfg.Namespace).Get(context.Background(), deployCfg.Name, metav1.GetOptions{})
	require.NoError(t, err)

	_, exists := actualDeploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	assert.True(t, exists)
}
