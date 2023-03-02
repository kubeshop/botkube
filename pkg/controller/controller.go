package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/status"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/notifier"
)

const (
	controllerStartMsg = "My watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "My watch has ended for cluster '%s'. See you soon!"

	finalMessageTimeout = 20 * time.Second
)

// ActionProvider defines a provider that is responsible for automated actions.
type ActionProvider interface {
	RenderedActionsForEvent(event event.Event, sourceBindings []string) ([]event.Action, error)
	ExecuteEventAction(ctx context.Context, action event.Action) interactive.CoreMessage
}

// Controller watches Kubernetes resources and send events to notifiers.
type Controller struct {
	log            logrus.FieldLogger
	conf           *config.Config
	notifiers      []notifier.Notifier
	statusReporter status.StatusReporter
}

// New create a new Controller instance.
func New(log logrus.FieldLogger, conf *config.Config, notifiers []notifier.Notifier, reporter status.StatusReporter) *Controller {
	return &Controller{
		log:            log,
		conf:           conf,
		notifiers:      notifiers,
		statusReporter: reporter,
	}
}

// Start creates new informer controllers to watch k8s resources
func (c *Controller) Start(ctx context.Context) error {
	c.log.Info("Starting controller...")

	c.log.Info("Sending welcome message...")
	err := notifier.SendPlaintextMessage(ctx, c.notifiers, fmt.Sprintf(controllerStartMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending first message: %w", err)
	}

	stopCh := ctx.Done()
	<-stopCh

	c.log.Info("Shutdown requested. Sending final message...")
	finalMsgCtx, cancelFn := context.WithTimeout(context.Background(), finalMessageTimeout)
	defer cancelFn()
	err = notifier.SendPlaintextMessage(finalMsgCtx, c.notifiers, fmt.Sprintf(controllerStopMsg, c.conf.Settings.ClusterName))
	if err != nil {
		return fmt.Errorf("while sending final message: %w", err)
	}

	// use separate ctx as parent ctx is already cancelled
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := c.statusReporter.ReportDeploymentShutdown(ctxTimeout); err != nil {
		return fmt.Errorf("while reporting botkube shutdown: %w", err)
	}

	return nil
}
