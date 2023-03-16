package insights

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/internal/status"
)

const infiniteRetry = 0

type K8sCollector struct {
	k8sCli                  kubernetes.Interface
	statusReporter          status.StatusReporter
	logger                  logrus.FieldLogger
	reportHeartbeatInterval int
	maxRetries              int
	failureCount            atomic.Int32
}

func NewK8sCollector(k8sCli kubernetes.Interface, reporter status.StatusReporter, logger logrus.FieldLogger, interval, maxRetries int) *K8sCollector {
	return &K8sCollector{k8sCli: k8sCli, statusReporter: reporter, logger: logger, reportHeartbeatInterval: interval, maxRetries: maxRetries}
}

// Start collects k8s insights, and it returns error once it cannot collect k8s node count.
func (k *K8sCollector) Start(ctx context.Context) error {
	err := retry.Do(
		func() error {
			list, err := k.k8sCli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				k.logger.Errorf("while getting node count: %w", err)
				k.failureCount.Add(1)
			} else {
				k.failureCount.Store(0)
				err = k.statusReporter.ReportHeartbeat(ctx, status.DeploymentHeartbeatInput{NodeCount: len(list.Items)})
				if err != nil {
					k.logger.Errorf("while reporting heartbeat: %w", err)
				}
			}
			if k.failureCount.Load() >= int32(k.maxRetries) {
				return retry.Unrecoverable(errors.New("reached maximum limit of node count retrieval"))
			}

			return retry.Error{} // This triggers retry, and with Attempts(0), it infinitely collects information until unrecoverable error.
		},
		retry.Delay(time.Duration(k.reportHeartbeatInterval)*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(infiniteRetry),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		return errors.Wrap(err, "while retrying")
	}
	return nil
}
