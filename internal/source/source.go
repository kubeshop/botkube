package source

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher watches for enabled sources events and send them to notifiers.
type Dispatcher struct {
	log       logrus.FieldLogger
	notifiers []notifier.Notifier
	manager   *plugin.Manager
	sources   []string
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers []notifier.Notifier, manager *plugin.Manager, sources []string) *Dispatcher {
	return &Dispatcher{
		log:       log,
		notifiers: notifiers,
		manager:   manager,
		sources:   sources,
	}
}

// Start starts all sources and dispatch received events.
func (d *Dispatcher) Start(ctx context.Context) error {
	for _, name := range d.sources {
		sourceClient, err := d.manager.GetSource(name)
		if err != nil {
			return fmt.Errorf("while getting source client for %s: %w", name, err)
		}
		out, err := sourceClient.Stream(ctx)
		if err != nil {
			return fmt.Errorf("while opening stream for %s: %w", name, err)
		}

		go func() {
			for {
				select {
				case event := <-out.Output:
					d.dispatch(ctx, event)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return nil
}

func (d *Dispatcher) dispatch(ctx context.Context, event []byte) {
	for _, n := range d.notifiers {
		go func(n notifier.Notifier) {
			err := n.SendMessageToAll(ctx, interactive.Message{
				Base: interactive.Base{
					Header: string(event),
				},
			})
			if err != nil {
				d.log.Errorf("while sending event: %s", err.Error())
			}
		}(n)
	}
}
