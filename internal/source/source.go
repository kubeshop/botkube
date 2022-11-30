package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher watches for enabled sources events and send them to notifiers.
type Dispatcher struct {
	log       logrus.FieldLogger
	notifiers []notifier.Notifier
	manager   *plugin.Manager
	cfg       *config.Config
	starter   func(ctx context.Context, pluginName string, pluginConfigs []any, sources []string) error
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers []notifier.Notifier, manager *plugin.Manager, cfg *config.Config) *Dispatcher {
	d := &Dispatcher{
		log:       log,
		notifiers: notifiers,
		manager:   manager,
		cfg:       cfg,
	}

	// TODO: it allows us to mock it for unit-test
	// will be removed once we will update the gRPC contract on source
	// and passing list of configuration.
	d.starter = d.start

	return d
}

// Start starts all sources and dispatch received events.
func (d *Dispatcher) Start(ctx context.Context) error {
	// startProcesses holds information about started unique plugin processes
	// We start a new plugin process each time we see a new order of source bindings.
	// We do that because we pass the array of configs to each `Stream` method and
	// the merging strategy for configs can depend on the order.
	// As a result our key is e.g. ['source-name1;source-name2']
	startProcesses := map[string]struct{}{}

	startPlugin := func(bindSources []string) error {
		key := strings.Join(bindSources, ";")

		_, found := startProcesses[key]
		if found {
			return nil // such configuration was already started
		}
		startProcesses[key] = struct{}{}

		// Holds the array of configs for a given plugin.
		// For example, ['botkube/kubernetes@v1.0.0']->[]{"cfg1", "cfg2"}
		sourcePluginConfigs := map[string][]any{}
		for _, sourceCfgGroupName := range bindSources {
			plugins := d.cfg.Sources[sourceCfgGroupName].Plugins
			for pluginName, pluginCfg := range plugins {
				if !pluginCfg.Enabled {
					continue
				}
				sourcePluginConfigs[pluginName] = append(sourcePluginConfigs[pluginName], pluginCfg.Config)
			}
		}

		for pluginName, configs := range sourcePluginConfigs {
			err := d.starter(ctx, pluginName, configs, bindSources)
			if err != nil {
				return fmt.Errorf("while starting plugin source %s: %w", pluginName, err)
			}
		}
		return nil
	}

	for _, commGroupCfg := range d.cfg.Communications {
		if commGroupCfg.Slack.Enabled {
			for _, channel := range commGroupCfg.Slack.Channels {
				if err := startPlugin(channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.SocketSlack.Enabled {
			for _, channel := range commGroupCfg.SocketSlack.Channels {
				if err := startPlugin(channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Mattermost.Enabled {
			for _, channel := range commGroupCfg.Mattermost.Channels {
				if err := startPlugin(channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Teams.Enabled {
			if err := startPlugin(commGroupCfg.Teams.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Discord.Enabled {
			for _, channel := range commGroupCfg.Discord.Channels {
				if err := startPlugin(channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Once we will have the gRPC contract established with proper Cloud Event schema, we should move also this logic here:
// https://github.com/kubeshop/botkube/blob/525c737956ff820a09321879284037da8bf5d647/pkg/controller/controller.go#L200-L253
func (d *Dispatcher) start(ctx context.Context, pluginName string, pluginConfigs []any, sources []string) error {
	log := d.log.WithFields(logrus.Fields{
		"pluginName": pluginName,
		"sources":    sources,
	})

	log.Info("Staring source streaming...")

	sourceClient, err := d.manager.GetSource(pluginName)
	if err != nil {
		return fmt.Errorf("while getting source client for %s: %w", pluginName, err)
	}

	// TODO(configure plugin): pass the `pluginConfigs`
	_ = pluginConfigs
	out, err := sourceClient.Stream(ctx)
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
