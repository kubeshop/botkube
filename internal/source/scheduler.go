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

// PluginDispatch holds information about source plugin.
type PluginDispatch struct {
	ctx                      context.Context
	pluginName               string
	pluginConfig             *source.Config
	sourceName               string
	sourceDisplayName        string
	isInteractivitySupported bool
	cfg                      *config.Config
	pluginContext            config.PluginContext
}

// ExternalRequestDispatch is a wrapper for PluginDispatch that holds the payload for external request.
type ExternalRequestDispatch struct {
	PluginDispatch
	payload []byte
}

// StartedSource holds information about started source plugin.
type StartedSource struct {
	SourceDisplayName        string
	PluginName               string
	PluginConfig             *source.Config
	IsInteractivitySupported bool
}

// Scheduler analyzes the provided configuration and based on that schedules plugin sources.
type Scheduler struct {
	log        logrus.FieldLogger
	cfg        *config.Config
	dispatcher pluginDispatcher

	// startedProcesses holds information about started unique plugin processes
	// We start a new plugin process each time we see a new order of source bindings.
	// We do that because we pass the array of configs to each `Stream` method and
	// the merging strategy for configs can depend on the order.
	// As a result our key is e.g. ['source-name1;source-name2']
	startedProcesses map[string]struct{}

	startedSourcePlugins map[string][]StartedSource
}

// NewScheduler create a new Scheduler instance.
func NewScheduler(log logrus.FieldLogger, cfg *config.Config, dispatcher pluginDispatcher) *Scheduler {
	return &Scheduler{
		log:                  log,
		cfg:                  cfg,
		dispatcher:           dispatcher,
		startedProcesses:     map[string]struct{}{},
		startedSourcePlugins: map[string][]StartedSource{},
	}
}

// Start starts all sources and dispatch received events.
func (d *Scheduler) Start(ctx context.Context) error {
	for _, commGroupCfg := range d.cfg.Communications {
		if commGroupCfg.CloudSlack.Enabled {
			for _, channel := range commGroupCfg.CloudSlack.Channels {
				if err := d.schedule(ctx, config.CloudSlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Slack.Enabled {
			for _, channel := range commGroupCfg.Slack.Channels {
				if err := d.schedule(ctx, config.SlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.SocketSlack.Enabled {
			for _, channel := range commGroupCfg.SocketSlack.Channels {
				if err := d.schedule(ctx, config.SocketSlackCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Mattermost.Enabled {
			for _, channel := range commGroupCfg.Mattermost.Channels {
				if err := d.schedule(ctx, config.MattermostCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Teams.Enabled {
			if err := d.schedule(ctx, config.TeamsCommPlatformIntegration.IsInteractive(), commGroupCfg.Teams.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Discord.Enabled {
			for _, channel := range commGroupCfg.Discord.Channels {
				if err := d.schedule(ctx, config.DiscordCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
					return err
				}
			}
		}

		if commGroupCfg.Webhook.Enabled {
			if err := d.schedule(ctx, false, commGroupCfg.Webhook.Bindings.Sources); err != nil {
				return err
			}
		}

		if commGroupCfg.Elasticsearch.Enabled {
			for _, index := range commGroupCfg.Elasticsearch.Indices {
				if err := d.schedule(ctx, false, index.Bindings.Sources); err != nil {
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
		if err := d.schedule(ctx, false, act.Bindings.Sources); err != nil {
			return err
		}
	}

	return nil
}

func (d *Scheduler) schedule(ctx context.Context, isInteractivitySupported bool, boundSources []string) error {
	for _, boundSource := range boundSources {
		err := d.schedulePlugin(ctx, isInteractivitySupported, boundSource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Scheduler) schedulePlugin(ctx context.Context, isInteractivitySupported bool, sourceName string) error {
	// As not all of our platforms supports interactivity, we need to schedule the same source twice. For example:
	//  - k8s-all-events_interactive/true
	//  - k8s-all-events_interactive/false
	// As a result each Stream method will know if it can produce interactive message or not.
	key := fmt.Sprintf("%s_interactive/%v", sourceName, isInteractivitySupported)

	_, found := d.startedProcesses[key]
	if found {
		d.log.Infof("Not starting %q as it was already started.", key)
		return nil // such configuration was already started
	}

	d.log.Infof("Starting a new stream for %q.", key)
	d.startedProcesses[key] = struct{}{}

	srcConfig, exists := d.cfg.Sources[sourceName]
	if !exists {
		return fmt.Errorf("source %q not found", sourceName)
	}
	for pluginName, pluginCfg := range srcConfig.Plugins {
		if !pluginCfg.Enabled {
			continue
		}
		// Unfortunately we need marshal it to get the raw data:
		// https://github.com/go-yaml/yaml/issues/13
		rawYAML, err := yaml.Marshal(pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("while marshaling config for %s from source %s : %w", pluginName, sourceName, err)
		}

		pluginConfig := &source.Config{
			RawYAML: rawYAML,
		}
		err = d.dispatcher.Dispatch(PluginDispatch{
			ctx:                      ctx,
			pluginName:               pluginName,
			pluginConfig:             pluginConfig,
			isInteractivitySupported: isInteractivitySupported,
			sourceName:               sourceName,
			sourceDisplayName:        srcConfig.DisplayName,
			cfg:                      d.cfg,
			pluginContext:            pluginCfg.Context,
		})
		if err != nil {
			return fmt.Errorf("while starting plugin source %s: %w", pluginName, err)
		}

		d.startedSourcePlugins[sourceName] = append(d.startedSourcePlugins[sourceName], StartedSource{
			SourceDisplayName:        srcConfig.DisplayName,
			PluginName:               pluginName,
			PluginConfig:             pluginConfig,
			IsInteractivitySupported: isInteractivitySupported,
		})
	}
	return nil
}

func (d *Scheduler) StartedSourcePlugins() map[string][]StartedSource {
	return d.startedSourcePlugins
}
