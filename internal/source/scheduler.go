package source

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
)

type pluginDispatcher interface {
	Dispatch(dispatch PluginDispatch) error
}

type PluginDispatch struct {
	ctx                      context.Context
	pluginName               string
	pluginConfigs            []*source.Config
	sourceName               string
	sourceDisplayName        string
	isInteractivitySupported bool
	cfg                      *config.Config
	pluginContext            config.PluginContext
}

// Scheduler analyzes the provided configuration and based on that schedules plugin sources.
type Scheduler struct {
	log            logrus.FieldLogger
	cfg            *config.Config
	dispatcher     pluginDispatcher
	dispatchConfig map[string]map[string]PluginDispatch
	schedulerChan  chan string

	// runningProcesses holds information about started unique plugin processes
	// We start a new plugin process each time we see a new order of source bindings.
	// We do that because we pass the array of configs to each `Stream` method and
	// the merging strategy for configs can depend on the order.
	// As a result our key is e.g. ['source-name1;source-name2']
	runningProcesses *processes
}

type processes struct {
	sync.RWMutex
	data map[string]struct{}
}

func (p *processes) add(key string) {
	p.Lock()
	defer p.Unlock()
	p.data[key] = struct{}{}
}

func (p *processes) delete(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.data, key)
}

func (p *processes) exists(key string) bool {
	p.RLock()
	defer p.RUnlock()
	_, ok := p.data[key]
	return ok
}

// NewScheduler create a new Scheduler instance.
func NewScheduler(ctx context.Context, log logrus.FieldLogger, cfg *config.Config, dispatcher pluginDispatcher, schedulerChan chan string) *Scheduler {
	s := &Scheduler{
		log:              log,
		cfg:              cfg,
		dispatcher:       dispatcher,
		runningProcesses: &processes{data: map[string]struct{}{}},
		dispatchConfig:   make(map[string]map[string]PluginDispatch),
		schedulerChan:    schedulerChan,
	}
	go s.monitorHealth(ctx)
	return s
}

func (d *Scheduler) monitorHealth(ctx context.Context) error {
	d.log.Info("Starting scheduler plugin health monitor")
	for {
		select {
		case <-ctx.Done():
			return nil
		case pluginName := <-d.schedulerChan:
			d.log.Infof("Scheduling restarted plugin %q", pluginName)
			d.runningProcesses.delete(pluginName)
			if err := d.schedule(ctx, pluginName); err != nil {
				d.log.Errorf("while scheduling %q: %s", pluginName, err)
			}
		}
	}
}

// Start starts all sources and dispatch received events.
func (d *Scheduler) Start(ctx context.Context) error {
	if err := d.generateConfigs(ctx); err != nil {
		return err
	}
	if err := d.schedule(ctx, ""); err != nil {
		return err
	}
	return nil
}

func (d *Scheduler) schedule(ctx context.Context, pluginFilter string) error {
	for _, sourceConfig := range d.dispatchConfig {
		for pluginName, config := range sourceConfig {
			if ok := d.runningProcesses.exists(pluginName); ok {
				d.log.Infof("Not starting %q as it was already started.", pluginName)
				continue
			}
			if pluginFilter != "" && pluginFilter != pluginName {
				d.log.Infof("Not starting %q as it doesn't pass plugin filter.", pluginName)
				continue
			}

			d.log.Infof("Starting a new stream for plugin %q", pluginName)
			if err := d.dispatcher.Dispatch(config); err != nil {
				return fmt.Errorf("while starting plugin source %s: %w", pluginName, err)
			}
			d.runningProcesses.add(pluginName)
		}
	}
	return nil
}

func (d *Scheduler) generateConfigs(ctx context.Context) error {
	for _, commGroupCfg := range d.cfg.Communications {
		if commGroupCfg.CloudSlack.Enabled {
			for _, channel := range commGroupCfg.CloudSlack.Channels {
				if err := d.generateSourceConfigs(ctx, config.CloudSlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Slack.Enabled {
			for _, channel := range commGroupCfg.Slack.Channels {
				if err := d.generateSourceConfigs(ctx, config.SlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.SocketSlack.Enabled {
			for _, channel := range commGroupCfg.SocketSlack.Channels {
				if err := d.generateSourceConfigs(ctx, config.SocketSlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Mattermost.Enabled {
			for _, channel := range commGroupCfg.Mattermost.Channels {
				if err := d.generateSourceConfigs(ctx, config.MattermostCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Teams.Enabled {
			if err := d.generateSourceConfigs(ctx, config.TeamsCommPlatformIntegration.IsInteractive(), commGroupCfg.Teams.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Discord.Enabled {
			for _, channel := range commGroupCfg.Discord.Channels {
				if err := d.generateSourceConfigs(ctx, config.DiscordCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Webhook.Enabled {
			if err := d.generateSourceConfigs(ctx, false, commGroupCfg.Webhook.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Elasticsearch.Enabled {
			for _, index := range commGroupCfg.Elasticsearch.Indices {
				if err := d.generateSourceConfigs(ctx, false, index.Bindings.Sources); err != nil {
					return err
				}
			}
		}
	}

	// Schedule all sources used by actions
	for _, act := range d.cfg.Actions {
		if !act.Enabled {
			continue
		}
		if err := d.generateSourceConfigs(ctx, false, act.Bindings.Sources); err != nil {
			return err
		}
	}

	return nil
}

func (d *Scheduler) generateSourceConfigs(ctx context.Context, isInteractivitySupported bool, bindSources []string) error {
	for _, bindSource := range bindSources {
		err := d.generatePluginConfig(ctx, isInteractivitySupported, bindSource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Scheduler) generatePluginConfig(ctx context.Context, isInteractivitySupported bool, sourceName string) error {
	// As not all of our platforms supports interactivity, we need to schedule the same source twice. For example:
	//  - botkube/kubernetes@1.0.0_interactive/true
	//  - botkube/kubernetes@1.0.0_interactive/false
	// As a result each Stream method will know if it can produce interactive message or not.
	key := fmt.Sprintf("%s_interactive/%v", sourceName, isInteractivitySupported)

	sourcePluginConfigs := map[string][]*source.Config{}
	srcConfig, exists := d.cfg.Sources[sourceName]
	if !exists {
		return fmt.Errorf("source %q not found", sourceName)
	}
	plugins := srcConfig.Plugins
	var pluginContext config.PluginContext
	for pluginName, pluginCfg := range plugins {
		if !pluginCfg.Enabled {
			continue
		}
		pluginContext = pluginCfg.Context
		// Unfortunately we need marshal it to get the raw data:
		// https://github.com/go-yaml/yaml/issues/13
		rawYAML, err := yaml.Marshal(pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("while marshaling config for %s from source %s : %w", pluginName, sourceName, err)
		}
		sourcePluginConfigs[pluginName] = append(sourcePluginConfigs[pluginName], &source.Config{
			RawYAML: rawYAML,
		})
	}

	for pluginName, configs := range sourcePluginConfigs {
		config := PluginDispatch{
			ctx:                      ctx,
			pluginName:               pluginName,
			pluginConfigs:            configs,
			isInteractivitySupported: isInteractivitySupported,
			sourceName:               sourceName,
			sourceDisplayName:        srcConfig.DisplayName,
			cfg:                      d.cfg,
			pluginContext:            pluginContext,
		}

		if dispatch := d.dispatchConfig[key]; dispatch == nil {
			d.dispatchConfig[key] = map[string]PluginDispatch{
				pluginName: config,
			}
		} else {
			dispatch[pluginName] = config
			d.dispatchConfig[key] = dispatch
		}

	}
	return nil
}
