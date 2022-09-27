package lifecycle

import (
	"context"
	"fmt"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"time"
)

const k8sDeploymentRestartPatchFmt = `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`

func newRestartHandler(log logrus.FieldLogger, k8sCli kubernetes.Interface, cfg config.LifecycleServer) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Infof(`Restarting the deployment "%s/%s"	...`, cfg.Deployment.Namespace, cfg.Deployment.Name)
		// This is what `kubectl rollout restart` does.
		restartData := fmt.Sprintf(k8sDeploymentRestartPatchFmt, time.Now().String())
		ctx := context.Background()
		_, err := k8sCli.AppsV1().Deployments(cfg.Deployment.Namespace).Patch(
			ctx,
			cfg.Deployment.Name,
			types.StrategicMergePatchType,
			[]byte(restartData),
			metav1.PatchOptions{FieldManager: "kubectl-rollout"},
		)
		if err != nil {
			errMsg := fmt.Sprintf("while restarting the Deployment: %s", err.Error())
			log.Error(errMsg)
			http.Error(writer, errMsg, http.StatusInternalServerError)
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(fmt.Sprintf(`Deployment "%s/%s" restarted successfully.`, cfg.Deployment.Namespace, cfg.Deployment.Name)))
	}
}
