package plugin

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/kubeshop/botkube/pkg/config"
)

// Collector provides functionality to collect all enabled plugins based on the Botkube configuration.
type Collector struct {
	log logrus.FieldLogger
}

// NewCollector returns a new Collector instance.
func NewCollector(log logrus.FieldLogger) *Collector {
	return &Collector{log: log}
}

// GetAllEnabledAndUsedPlugins returns the list of all plugins that are both enabled and bind to at
// least one communicator or action (automation) that is enabled.
func (c *Collector) GetAllEnabledAndUsedPlugins(cfg *config.Config) ([]string, []string) {
	var (
		bindExecutors = map[string]struct{}{}
		bindSources   = map[string]struct{}{}
	)

	// Collect all used executors/sources by communication platforms
	collect := func(channels config.IdentifiableMap[config.ChannelBindingsByName]) {
		for _, bindings := range channels {
			for _, name := range bindings.Bindings.Executors {
				bindExecutors[name] = struct{}{}
			}
			for _, name := range bindings.Bindings.Sources {
				bindSources[name] = struct{}{}
			}
		}
	}

	for _, commGroupCfg := range cfg.Communications {
		if commGroupCfg.Slack.Enabled {
			collect(commGroupCfg.Slack.Channels)
		}

		if commGroupCfg.SocketSlack.Enabled {
			collect(commGroupCfg.SocketSlack.Channels)
		}

		if commGroupCfg.Mattermost.Enabled {
			collect(commGroupCfg.Mattermost.Channels)
		}

		if commGroupCfg.Teams.Enabled {
			for _, name := range commGroupCfg.Teams.Bindings.Executors {
				bindExecutors[name] = struct{}{}
			}
			for _, name := range commGroupCfg.Teams.Bindings.Sources {
				bindSources[name] = struct{}{}
			}
		}

		if commGroupCfg.Discord.Enabled {
			for _, bindings := range commGroupCfg.Discord.Channels {
				for _, name := range bindings.Bindings.Executors {
					bindExecutors[name] = struct{}{}
				}
				for _, name := range bindings.Bindings.Sources {
					bindSources[name] = struct{}{}
				}
			}
		}

		if commGroupCfg.Webhook.Enabled {
			for _, name := range commGroupCfg.Webhook.Bindings.Sources {
				bindSources[name] = struct{}{}
			}
		}

		if commGroupCfg.Elasticsearch.Enabled {
			for _, index := range commGroupCfg.Elasticsearch.Indices {
				for _, name := range index.Bindings.Sources {
					bindSources[name] = struct{}{}
				}
			}
		}
	}

	// Collect all used executors/sources by actions
	for _, act := range cfg.Actions {
		if !act.Enabled {
			continue
		}
		for _, executorCfgName := range act.Bindings.Executors {
			bindExecutors[executorCfgName] = struct{}{}
		}
		for _, sourceCfgName := range act.Bindings.Sources {
			bindSources[sourceCfgName] = struct{}{}
		}
	}

	// Collect all executors that are both enabled and bind to at least one communicator or action (automation) that is enabled..
	usedExecutorPlugins := map[string]struct{}{}
	for groupName, groupItems := range cfg.Executors {
		for name, executor := range groupItems.Plugins {
			l := c.log.WithFields(logrus.Fields{
				"groupName": groupName,
				"pluginKey": name,
			})

			if !executor.Enabled {
				l.Debug("Executor plugin defined but not enabled.")
				continue
			}

			_, found := bindExecutors[groupName]
			if !found {
				l.Debug("Executor plugin defined and enabled but not used by any platform or standalone action")
				continue
			}

			l.Debug("Marking executor plugin as enabled")
			usedExecutorPlugins[name] = struct{}{}
		}
	}

	// Collect all sources that are both enabled and bind to at least one communicator that is enabled.
	usedSourcePlugins := map[string]struct{}{}
	for groupName, groupItems := range cfg.Sources {
		for name, source := range groupItems.Plugins {
			l := c.log.WithFields(logrus.Fields{
				"groupName": groupName,
				"pluginKey": name,
			})

			if !source.Enabled {
				l.Debug("Source plugin defined but not enabled.")

				continue
			}
			_, found := bindSources[groupName]
			if !found {
				l.Debug("Source plugin defined and enabled but not used by any platform or standalone action")
				continue
			}

			l.Debug("Marking source plugin as enabled")
			usedSourcePlugins[name] = struct{}{}
		}
	}

	return maps.Keys(usedExecutorPlugins), maps.Keys(usedSourcePlugins)
}
