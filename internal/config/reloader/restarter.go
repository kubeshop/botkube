package reloader

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	k8sDeploymentRestartPatchFmt = `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`
	reloadMsgFmt                 = ":arrows_counterclockwise: Configuration reload requested for cluster '%s'. Hold on a sec..."
	fieldManagerName             = "botkube"
)

// SendMessageFn defines a function which sends a given message.
type SendMessageFn func(msg string) error

// Restarter is responsible for restarting the deployment.
type Restarter struct {
	log         logrus.FieldLogger
	k8sCli      kubernetes.Interface
	deploy      config.K8sResourceRef
	clusterName string
	sendMsgFn   SendMessageFn
}

// NewRestarter returns new Restarter.
func NewRestarter(log logrus.FieldLogger, k8sCli kubernetes.Interface, deploy config.K8sResourceRef, clusterName string, sendMsgFn SendMessageFn) *Restarter {
	return &Restarter{
		log:         log,
		k8sCli:      k8sCli,
		deploy:      deploy,
		clusterName: clusterName,
		sendMsgFn:   sendMsgFn,
	}
}

// Do restarts the deployment.
func (r *Restarter) Do(ctx context.Context) error {
	r.log.Info("Reload requested. Sending last message before exit...")
	err := r.sendMsgFn(fmt.Sprintf(reloadMsgFmt, r.clusterName))
	if err != nil {
		errMsg := fmt.Sprintf("while sending last message: %s", err.Error())
		r.log.Errorf(errMsg)
		// continue anyway, this is a non-blocking error
	}

	r.log.Infof(`Reloading te the deployment "%s/%s"...`, r.deploy.Namespace, r.deploy.Name)
	// This is what `kubectl rollout restart` does.
	restartData := fmt.Sprintf(k8sDeploymentRestartPatchFmt, time.Now().String())
	_, err = r.k8sCli.AppsV1().Deployments(r.deploy.Namespace).Patch(
		ctx,
		r.deploy.Name,
		types.StrategicMergePatchType,
		[]byte(restartData),
		metav1.PatchOptions{FieldManager: fieldManagerName},
	)
	if err != nil {
		return fmt.Errorf("while restarting the Deployment: %w", err)
	}
	r.log.Infof(`Restarting deployment "%s/%s" ended successfully.`, r.deploy.Namespace, r.deploy.Name)
	return nil
}
