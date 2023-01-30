package execute

import (
	"context"
	"fmt"
	"github.com/kubeshop/botkube/pkg/bot/interactive"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
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
		"bindings": bindings,
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

	resp, err := cli.Execute(ctx, executor.ExecuteInput{
		Command: command,
		Configs: configs,
	})
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return "", NewExecutionCommandError(err.Error())
		}
		return "", NewExecutionCommandError(s.Message())
	}

	return resp.Data, nil
}

func (e *PluginExecutor) Help(ctx context.Context, bindings []string, args []string, command string) (interactive.Message, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": bindings,
		"command":  command,
	}).Debugf("Handling plugin help command...")

	cmdName := args[0]
	_, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while getting concrete plugin client: %w", err)
	}
	e.log.Debug("running help command")
	return cli.Help(ctx)
}

func (e *PluginExecutor) collectConfigs(plugins []config.Plugin) ([]*executor.Config, error) {
	var configs []*executor.Config

	for _, plugin := range plugins {
		if plugin.Config == nil {
			continue
		}

		// Unfortunately we need marshal it to get the raw data:
		// https://github.com/go-yaml/yaml/issues/13
		raw, err := yaml.Marshal(plugin.Config)
		if err != nil {
			return nil, err
		}

		configs = append(configs, &executor.Config{
			RawYAML: raw,
		})
	}

	return configs, nil
}

func (e *PluginExecutor) getEnabledPlugins(bindings []string, cmdName string) ([]config.Plugin, string) {
	var (
		out            []config.Plugin
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

			_, pluginName, _, _ := config.DecomposePluginKey(pluginKey)
			if pluginName != cmdName {
				continue
			}

			fullPluginName = pluginKey
			out = append(out, pluginDetails)
		}
	}

	return out, fullPluginName
}
