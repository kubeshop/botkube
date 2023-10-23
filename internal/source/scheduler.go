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

const (
	emptyPluginFilter = ""
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
	incomingWebhook          IncomingWebhookData
}

// ExternalRequestDispatch is a wrapper for PluginDispatch that holds the payload for external request.
type ExternalRequestDispatch struct {
	PluginDispatch
	payload []byte
}

// StartedSources holds information about started source plugins grouped by interactivity supported.
type StartedSources map[bool]StartedSource

// StartedSource holds information about started source plugin.
type StartedSource struct {
	SourceDisplayName        string
	PluginName               string
	PluginConfig             *source.Config
	IsInteractivitySupported bool
}

// Scheduler analyzes the provided configuration and based on that schedules plugin sources.
type Scheduler struct {
	log            logrus.FieldLogger
	cfg            *config.Config
	dispatcher     pluginDispatcher
	dispatchConfig map[string]map[string]PluginDispatch
	schedulerChan  chan string

	openedStreams        *openedStreams
	startedSourcePlugins map[string]StartedSources
}

type openedStreams struct {
	sync.RWMutex
	data map[string]map[string]struct{}
}

func (p *openedStreams) reportStartedStreamWithConfiguration(plugin, configuration string) {
	p.Lock()
	defer p.Unlock()
	if p.data == nil {
		p.data = map[string]map[string]struct{}{}
	}

	if p.data[plugin] == nil {
		p.data[plugin] = map[string]struct{}{}
	}

	p.data[plugin][configuration] = struct{}{}
}

func (p *openedStreams) deleteAllStartedStreamsForPlugin(plugin string) {
	p.Lock()
	defer p.Unlock()
	delete(p.data, plugin)
}

func (p *openedStreams) isStartedStreamWithConfiguration(plugin, configuration string) bool {
	p.RLock()
	defer p.RUnlock()
	_, ok := p.data[plugin][configuration]
	return ok
}

// NewScheduler create a new Scheduler instance.
func NewScheduler(ctx context.Context, log logrus.FieldLogger, cfg *config.Config, dispatcher pluginDispatcher, schedulerChan chan string) *Scheduler {
	s := &Scheduler{
		log:                  log,
		cfg:                  cfg,
		dispatcher:           dispatcher,
		startedSourcePlugins: map[string]StartedSources{},
		openedStreams:        &openedStreams{data: map[string]map[string]struct{}{}},
		dispatchConfig:       make(map[string]map[string]PluginDispatch),
		schedulerChan:        schedulerChan,
	}
	go s.monitorHealth(ctx)
	return s
}

// Start starts all sources and dispatch received events.
func (d *Scheduler) Start(ctx context.Context) error {
	if err := d.generateConfigs(ctx); err != nil {
		return fmt.Errorf("while generating configs for sources: %w", err)
	}
	if err := d.schedule(emptyPluginFilter); err != nil {
		return fmt.Errorf("while scheduling source dispatch: %w", err)
	}
	return nil
}

func (d *Scheduler) monitorHealth(ctx context.Context) {
	d.log.Info("Starting scheduler plugin health monitor")
	for {
		select {
		case <-ctx.Done():
			return
		case pluginName := <-d.schedulerChan:
			d.log.Debugf("Scheduling restarted plugin %q", pluginName)
			//  botkube/kubernetes map to source configuration ->
			// 	   - k8s-all-events_interactive/true
			//	   - k8s-all-events_interactive/false
			d.openedStreams.deleteAllStartedStreamsForPlugin(pluginName)
			if err := d.schedule(pluginName); err != nil {
				d.log.Errorf("while scheduling %q: %s", pluginName, err)
			}
		}
	}
}

func (d *Scheduler) schedule(pluginFilter string) error {
	for configKey, sourceConfig := range d.dispatchConfig {
		for pluginName, config := range sourceConfig {
			if pluginFilter != emptyPluginFilter && pluginFilter != pluginName {
				d.log.Debugf("Not starting %q as it doesn't pass plugin filter.", pluginName)
				continue
			}

			if ok := d.openedStreams.isStartedStreamWithConfiguration(pluginName, configKey); ok {
				d.log.Infof("Not starting %q as it was already started.", pluginName)
				continue
			}

			d.log.Infof("Starting a new stream for plugin %q", pluginName)
			if err := d.dispatcher.Dispatch(config); err != nil {
				return fmt.Errorf("while starting plugin source %s: %w", pluginName, err)
			}

			d.openedStreams.reportStartedStreamWithConfiguration(pluginName, configKey)
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
		if commGroupCfg.CloudTeams.Enabled {
			for _, teams := range commGroupCfg.CloudTeams.Teams {
				for _, channel := range teams.Channels {
					if err := d.generateSourceConfigs(ctx, config.CloudTeamsCommPlatformIntegration.IsInteractive(), channel.Bindings.Sources); err != nil {
						return err
					}
				}
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

func (d *Scheduler) generateSourceConfigs(ctx context.Context, isInteractivitySupported bool, boundSources []string) error {
	for _, boundSource := range boundSources {
		err := d.generatePluginConfig(ctx, isInteractivitySupported, boundSource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Scheduler) generatePluginConfig(ctx context.Context, isInteractivitySupported bool, sourceName string) error {
	// As not all of our platforms supports interactivity, we need to schedule the same source twice. For example:
	//  - k8s-all-events_interactive/true
	//  - k8s-all-events_interactive/false
	// As a result, each Stream method will know if it can produce an interactive message or not.
	key := fmt.Sprintf("%s_interactive/%v", sourceName, isInteractivitySupported)

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
		config := PluginDispatch{
			ctx:                      ctx,
			pluginName:               pluginName,
			pluginConfig:             pluginConfig,
			isInteractivitySupported: isInteractivitySupported,
			sourceName:               sourceName,
			sourceDisplayName:        srcConfig.DisplayName,
			cfg:                      d.cfg,
			pluginContext:            pluginCfg.Context,
			incomingWebhook: IncomingWebhookData{
				inClusterBaseURL: d.cfg.Plugins.IncomingWebhook.InClusterBaseURL,
			},
		}

		d.dispatchConfig[key] = map[string]PluginDispatch{
			pluginName: config,
		}

		if _, exists := d.startedSourcePlugins[config.sourceName]; !exists {
			d.startedSourcePlugins[config.sourceName] = StartedSources{}
		}

		d.startedSourcePlugins[config.sourceName][config.isInteractivitySupported] = StartedSource{
			SourceDisplayName:        config.sourceDisplayName,
			PluginName:               pluginName,
			PluginConfig:             config.pluginConfig,
			IsInteractivitySupported: config.isInteractivitySupported,
		}
	}
	return nil
}

func (d *Scheduler) StartedSourcePlugins() map[string]StartedSources {
	return d.startedSourcePlugins
}
