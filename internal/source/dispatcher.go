package source

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/audit"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher provides functionality to starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
type Dispatcher struct {
	log           logrus.FieldLogger
	notifiers     []notifier.Notifier
	manager       *plugin.Manager
	auditReporter audit.AuditReporter
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers []notifier.Notifier, manager *plugin.Manager, auditReporter audit.AuditReporter) *Dispatcher {
	return &Dispatcher{
		log:           log,
		notifiers:     notifiers,
		manager:       manager,
		auditReporter: auditReporter,
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
				d.dispatch(ctx, event, sources, pluginName)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (d *Dispatcher) dispatch(ctx context.Context, event []byte, sources []string, pluginName string) {
	for _, n := range d.notifiers {
		go func(n notifier.Notifier) {
			eventString := string(event)
			msg := interactive.CoreMessage{
				Description: eventString,
			}
			if err := n.SendMessage(ctx, msg, sources); err != nil {
				d.log.Errorf("while sending event: %s", err.Error())
			}
			if err := d.reportAudit(ctx, pluginName, eventString, sources); err != nil {
				d.log.Errorf("while reporting audit event: %s", err.Error())
			}
		}(n)
	}
}

func (d *Dispatcher) reportAudit(ctx context.Context, pluginName, event string, sources []string) error {
	e := audit.SourceAuditEvent{
		CreatedAt:  time.Now().Format(time.RFC3339),
		PluginName: pluginName,
		Event:      event,
		Bindings:   sources,
	}
	return d.auditReporter.ReportSourceAuditEvent(ctx, e)
}
