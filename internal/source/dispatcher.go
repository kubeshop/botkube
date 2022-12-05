package source

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher provides functionality to starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
type Dispatcher struct {
	log       logrus.FieldLogger
	notifiers []notifier.Notifier
	manager   *plugin.Manager
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers []notifier.Notifier, manager *plugin.Manager) *Dispatcher {
	return &Dispatcher{
		log:       log,
		notifiers: notifiers,
		manager:   manager,
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

	log.Info("Staring source streaming...")

	sourceClient, err := d.manager.GetSource(pluginName)
	if err != nil {
		return fmt.Errorf("while getting source client for %s: %w", pluginName, err)
	}

	out, err := sourceClient.Stream(ctx, pluginConfigs)
	if err != nil {
		return fmt.Errorf("while opening stream for %s: %w", pluginName, err)
	}

	go func() {
		for {
			select {
			case event := <-out.Output:
				log.WithField("event", string(event)).Debug("Dispatching received event...")
				d.dispatch(ctx, event, sources)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (d *Dispatcher) dispatch(ctx context.Context, event []byte, sources []string) {
	for _, n := range d.notifiers {
		go func(n notifier.Notifier) {
			msg := interactive.Message{
				Base: interactive.Base{
					Description: string(event),
				},
			}
			err := n.SendGenericMessage(ctx, &genericMessage{response: msg}, sources)
			if err != nil {
				d.log.Errorf("while sending event: %s", err.Error())
			}
		}(n)
	}
}

type genericMessage struct {
	response interactive.Message
}

// ForBot returns interactive message.
func (g *genericMessage) ForBot(string) interactive.Message {
	return g.response
}
