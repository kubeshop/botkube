package source

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher provides functionality to starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
type Dispatcher struct {
	log            logrus.FieldLogger
	notifiers      []notifier.Notifier
	manager        *plugin.Manager
	actionProvider ActionProvider
	reporter       AnalyticsReporter
}

// ActionProvider defines a provider that is responsible for automated actions.
type ActionProvider interface {
	RenderedActions(data any, sourceBindings []string) ([]event.Action, error)
	ExecuteAction(ctx context.Context, action event.Action) interactive.CoreMessage
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportHandledEventSuccess reports a successfully handled event using a given communication platform.
	ReportHandledEventSuccess(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails analytics.EventDetails) error

	// ReportHandledEventError reports a failure while handling event using a given communication platform.
	ReportHandledEventError(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails analytics.EventDetails, err error) error

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers []notifier.Notifier, manager *plugin.Manager, actionProvider ActionProvider, reporter AnalyticsReporter) *Dispatcher {
	return &Dispatcher{
		log:            log,
		notifiers:      notifiers,
		manager:        manager,
		actionProvider: actionProvider,
		reporter:       reporter,
	}
}

// Dispatch starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
// Once we will have the gRPC contract established with proper Cloud Event schema, we should move also this logic here:
// https://github.com/kubeshop/botkube/blob/525c737956ff820a09321879284037da8bf5d647/pkg/controller/controller.go#L200-L253
func (d *Dispatcher) Dispatch(ctx context.Context, pluginName string, pluginConfigs []*source.Config, sources []string) error {
	log := d.log.WithFields(logrus.Fields{
		"pluginName": pluginName,
		"sources":    sources,
	})

	log.Info("Start source streaming...")

	sourceClient, err := d.manager.GetSource(pluginName)
	if err != nil {
		return fmt.Errorf("while getting source client for %s: %w", pluginName, err)
	}

	out, err := sourceClient.Stream(ctx, source.StreamInput{
		Configs: pluginConfigs,
	})
	if err != nil {
		return fmt.Errorf("while opening stream for %s: %w", pluginName, err)
	}

	go func() {
		for {
			select {
			case event := <-out.Output:
				log.WithField("event", string(event)).Debug("Dispatching received event...")
				d.dispatch(ctx, event, sources)
			case msg := <-out.Message:
				log.WithField("message", msg).Debug("Dispatching received message...")
				d.dispatchMsg(ctx, msg, sources)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (d *Dispatcher) dispatchMsg(ctx context.Context, message source.Message, sources []string) {
	for _, n := range d.notifiers {
		go func(n notifier.Notifier) {
			msg := interactive.CoreMessage{
				Message: message.Data,
			}
			err := n.SendMessage(ctx, msg, sources)
			if err != nil {
				d.log.Errorf("while sending event: %s", err.Error())
			}
		}(n)
	}

	// execute actions
	actions, err := d.actionProvider.RenderedActions(message.Metadata, sources)
	if err != nil {
		d.log.Errorf("while rendering automated actions: %s", err.Error())
		return
	}
	for _, act := range actions {
		d.log.Infof("Executing action %q (command: %q)...", act.DisplayName, act.Command)
		genericMsg := d.actionProvider.ExecuteAction(ctx, act)
		for _, n := range d.notifiers {
			go func(n notifier.Notifier) {
				defer analytics.ReportPanicIfOccurs(d.log, d.reporter)
				err := n.SendMessage(ctx, genericMsg, sources)
				if err != nil {
					d.log.Errorf("while sending event: %s", err.Error())
				}
			}(n)
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, event []byte, sources []string) {
	for _, n := range d.notifiers {
		go func(n notifier.Notifier) {
			msg := interactive.CoreMessage{
				Description: string(event),
			}
			err := n.SendMessage(ctx, msg, sources)
			if err != nil {
				d.log.Errorf("while sending event: %s", err.Error())
			}
		}(n)
	}
}
