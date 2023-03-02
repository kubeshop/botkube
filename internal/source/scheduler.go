package source

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
)

type pluginDispatcher interface {
	Dispatch(dispatch PluginDispatch) error
}

type PluginDispatch struct {
	ctx           context.Context
	pluginName    string
	pluginConfigs []*source.Config
	sources       []string
	cfg           *config.Config
}

// Scheduler analyzes the provided configuration and based on that schedules plugin sources.
type Scheduler struct {
	log        logrus.FieldLogger
	cfg        *config.Config
	dispatcher pluginDispatcher

	// startProcesses holds information about started unique plugin processes
	// We start a new plugin process each time we see a new order of source bindings.
	// We do that because we pass the array of configs to each `Stream` method and
	// the merging strategy for configs can depend on the order.
	// As a result our key is e.g. ['source-name1;source-name2']
	startProcesses map[string]struct{}
}

// NewScheduler create a new Scheduler instance.
func NewScheduler(log logrus.FieldLogger, cfg *config.Config, dispatcher pluginDispatcher) *Scheduler {
	return &Scheduler{
		log:            log,
		cfg:            cfg,
		dispatcher:     dispatcher,
		startProcesses: map[string]struct{}{},
	}
}

// Start starts all sources and dispatch received events.
func (d *Scheduler) Start(ctx context.Context) error {
	for _, commGroupCfg := range d.cfg.Communications {
		if commGroupCfg.Slack.Enabled {
			for _, channel := range commGroupCfg.Slack.Channels {
				if err := d.schedule(ctx, channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.SocketSlack.Enabled {
			for _, channel := range commGroupCfg.SocketSlack.Channels {
				if err := d.schedule(ctx, channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Mattermost.Enabled {
			for _, channel := range commGroupCfg.Mattermost.Channels {
				if err := d.schedule(ctx, channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Teams.Enabled {
			if err := d.schedule(ctx, commGroupCfg.Teams.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Discord.Enabled {
			for _, channel := range commGroupCfg.Discord.Channels {
				if err := d.schedule(ctx, channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (d *Scheduler) schedule(ctx context.Context, bindSources []string) error {
	for _, bindSource := range bindSources {
		err := d.schedulePlugin(ctx, bindSource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Scheduler) schedulePlugin(ctx context.Context, key string) error {
	_, found := d.startProcesses[key]
	if found {
		return nil // such configuration was already started
	}
	d.startProcesses[key] = struct{}{}

	// Holds the array of configs for a given plugin. Currently, we have only one key for backward compatibility
	// For example, ['botkube/kubernetes@v1.0.0']->[]{"cfg1"}
	sourcePluginConfigs := map[string][]*source.Config{}
	plugins := d.cfg.Sources[key].Plugins
	for pluginName, pluginCfg := range plugins {
		if !pluginCfg.Enabled {
			continue
		}

		// Unfortunately we need marshal it to get the raw data:
		// https://github.com/go-yaml/yaml/issues/13
		rawYAML, err := yaml.Marshal(pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("while marshaling config for %s from source %s : %w", pluginName, key, err)
		}
		sourcePluginConfigs[pluginName] = append(sourcePluginConfigs[pluginName], &source.Config{
			RawYAML: rawYAML,
		})
	}

	for pluginName, configs := range sourcePluginConfigs {
		err := d.dispatcher.Dispatch(PluginDispatch{
			ctx:           ctx,
			pluginName:    pluginName,
			pluginConfigs: configs,
			sources:       []string{key},
			cfg:           d.cfg,
		})
		if err != nil {
			return fmt.Errorf("while starting plugin source %s: %w", pluginName, err)
		}
	}
	return nil
}
