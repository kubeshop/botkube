package plugin

import (
	"github.com/kubeshop/botkube/pkg/config"
)

// GetAllEnabledAndUsedPlugins returns the list of all plugins that are both enabled and bind to at
// least one communicator that is also enabled.
func GetAllEnabledAndUsedPlugins(cfg *config.Config) []string {
	// Collect all used executor
	bindExecutors := map[string]struct{}{}

	collect := func(channels config.IdentifiableMap[config.ChannelBindingsByName]) {
		for _, bindings := range channels {
			for _, name := range bindings.Bindings.Executors {
				bindExecutors[name] = struct{}{}
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
		}

		if commGroupCfg.Discord.Enabled {
			for _, bindings := range commGroupCfg.Discord.Channels {
				for _, name := range bindings.Bindings.Executors {
					bindExecutors[name] = struct{}{}
				}
			}
		}
	}

	// Collect all executors that are both enabled and bind to at least one communicator that is enabled.
	var usedExecutorPlugins []string
	for groupName, groupItems := range cfg.Executors {
		for name, executor := range groupItems.Plugins {
			if !executor.Enabled {
				continue
			}
			_, found := bindExecutors[groupName]
			if !found {
				continue
			}

			usedExecutorPlugins = append(usedExecutorPlugins, name)
		}
	}

	return usedExecutorPlugins
}
