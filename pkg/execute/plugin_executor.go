package execute

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
)

// PluginExecutor provides functionality to run registered Botkube plugins.
type PluginExecutor struct {
	log           logrus.FieldLogger
	cfg           config.Config
	pluginManager *plugin.Manager
}

// NewPluginExecutor creates a new instance of PluginExecutor.
func NewPluginExecutor(log logrus.FieldLogger, cfg config.Config, manager *plugin.Manager) *PluginExecutor {
	return &PluginExecutor{
		log:           log,
		cfg:           cfg,
		pluginManager: manager,
	}
}

// CanHandle returns true if it's a known plugin executor.
func (e *PluginExecutor) CanHandle(bindings []string, args []string) bool {
	if len(args) == 0 {
		return false
	}

	cmdName := args[0]
	plugins, _ := e.getEnabledPlugins(bindings, cmdName)

	return len(plugins) > 0
}

// GetCommandPrefix gets verb command with k8s alias prefix.
func (e *PluginExecutor) GetCommandPrefix(args []string) string {
	if len(args) == 0 {
		return ""
	}

	return args[0]
}

// Execute executes plugin executor based on a given command.
func (e *PluginExecutor) Execute(ctx context.Context, bindings []string, args []string, command string) (string, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": command,
		"command":  command,
	}).Debugf("Handling plugin command...")

	cmdName := args[0]
	plugins, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	configs, err := e.collectConfigs(plugins)
	if err != nil {
		return "", fmt.Errorf("while collecting configs: %w", err)
	}

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return "", fmt.Errorf("while getting concrete plugin client: %w", err)
	}

	// TODO(configure plugin): Execute RPC with all data:
	//  - command
	//  - all configuration but in proper order (so it can be merged properly)
	_ = configs
	resp, err := cli.Execute(ctx, &executor.ExecuteRequest{Command: command})
	if err != nil {
		return "", fmt.Errorf("while executing gRPC call: %w", err)
	}

	return resp.Data, nil
}

func (e *PluginExecutor) collectConfigs(plugins []config.PluginExecutor) ([]string, error) {
	var configs []string

	for _, plugin := range plugins {
		if plugin.Config == nil {
			continue
		}

		raw, err := yaml.Marshal(plugin.Config)
		if err != nil {
			return nil, err
		}

		configs = append(configs, string(raw))
	}

	return configs, nil
}

func (e *PluginExecutor) getEnabledPlugins(bindings []string, cmdName string) ([]config.PluginExecutor, string) {
	var (
		out            []config.PluginExecutor
		fullPluginName string
	)

	for _, bindingName := range bindings {
		bindExecutors, found := e.cfg.Executors[bindingName]
		if !found {
			continue
		}

		for pluginKey, pluginDetails := range bindExecutors.Plugins {
			if !pluginDetails.Enabled {
				continue
			}

			_, pluginName, _, _ := plugin.DecomposePluginKey(pluginKey)
			if pluginName != cmdName {
				continue
			}

			fullPluginName = pluginKey
			out = append(out, pluginDetails)
		}
	}

	return out, fullPluginName
}
