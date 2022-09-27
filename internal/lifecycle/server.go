package lifecycle

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"time"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/httpsrv"
)

const (
	k8sDeploymentRestartPatchFmt = `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`
	reloadMsgFmt                 = "Configuration reload requested for cluster '%s'. I shall halt my watch till I read it."
)

type SendMessageFn func(msg string) error

func NewServer(log logrus.FieldLogger, k8sCli kubernetes.Interface, cfg config.LifecycleServer, clusterName string, sendMsgFn SendMessageFn) *httpsrv.Server {
	addr := fmt.Sprintf(":%d", cfg.Port)
	router := mux.NewRouter()
	reloadHandler := newReloadHandler(log, k8sCli, cfg.Deployment, clusterName, sendMsgFn)
	router.HandleFunc("/reload", reloadHandler)
	return httpsrv.New(log, addr, router)
}

func newReloadHandler(log logrus.FieldLogger, k8sCli kubernetes.Interface, deploy config.K8sResourceRef, clusterName string, sendMsgFn SendMessageFn) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Info("Reload requested. Sending last message before exit...")
		err := sendMsgFn(fmt.Sprintf(reloadMsgFmt, clusterName))
		if err != nil {
			errMsg := fmt.Sprintf("while sending last message: %s", err.Error())
			log.Errorf(errMsg)

			// continue anyway, this is a non-blocking error
		}

		log.Infof(`Reloading te the deployment "%s/%s"...`, deploy.Namespace, deploy.Name)
		// This is what `kubectl rollout restart` does.
		restartData := fmt.Sprintf(k8sDeploymentRestartPatchFmt, time.Now().String())
		ctx := request.Context()
		_, err = k8sCli.AppsV1().Deployments(deploy.Namespace).Patch(
			ctx,
			deploy.Name,
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
		writer.Write([]byte(fmt.Sprintf(`Deployment "%s/%s" restarted successfully.`, deploy.Namespace, deploy.Name)))
	}
}
