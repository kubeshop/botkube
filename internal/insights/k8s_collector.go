package insights

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/internal/heartbeat"
)

type K8sCollector struct {
	k8sCli                  kubernetes.Interface
	heartbeatReporter       heartbeat.HeartbeatReporter
	logger                  logrus.FieldLogger
	reportHeartbeatInterval int
	maxRetries              int
	failureCount            atomic.Int32
}

func NewK8sCollector(k8sCli kubernetes.Interface, reporter heartbeat.HeartbeatReporter, logger logrus.FieldLogger, interval, maxRetries int) *K8sCollector {
	return &K8sCollector{k8sCli: k8sCli, heartbeatReporter: reporter, logger: logger, reportHeartbeatInterval: interval, maxRetries: maxRetries}
}

// Start collects k8s insights, and it returns error once it cannot collect k8s node count.
func (k *K8sCollector) Start(ctx context.Context, remoteCfgEnabled bool) error {
	if !remoteCfgEnabled {
		k.logger.Debug("Remote config is not enabled, skipping k8s insights collection...")
		return nil
	}
	ticker := time.NewTicker(time.Duration(k.reportHeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			k.logger.Info("Shutdown requested. Finishing...")
			return nil
		case <-ticker.C:
			k.logger.Debug("Collecting Kubernetes insights")
			list, err := k.k8sCli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				k.logger.Errorf("while getting node count: %w", err)
				k.failureCount.Add(1)
			} else {
				k.failureCount.Store(0)
				err = k.heartbeatReporter.ReportHeartbeat(ctx, heartbeat.DeploymentHeartbeatInput{NodeCount: len(list.Items)})
				if err != nil {
					k.logger.Errorf("while reporting heartbeat: %w", err)
				}
			}
			if k.failureCount.Load() >= int32(k.maxRetries) {
				return errors.New("reached maximum limit of node count retrieval")
			}
		}
	}
}
