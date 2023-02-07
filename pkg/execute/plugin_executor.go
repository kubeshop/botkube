package execute

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
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
func (e *PluginExecutor) Execute(ctx context.Context, bindings []string, cmdCtx CommandContext) (interactive.Message, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": bindings,
		"command":  cmdCtx.CleanCmd,
	}).Debugf("Handling plugin command...")

	cmdName := cmdCtx.Args[0]
	plugins, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	configs, err := e.collectConfigs(plugins)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while collecting configs: %w", err)
	}

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while getting concrete plugin client: %w", err)
	}

	resp, err := cli.Execute(ctx, executor.ExecuteInput{
		Command: cmdCtx.CleanCmd,
		Configs: configs,
		Context: executor.ExecuteInputContext{
			CommunicationPlatform: cmdCtx.Platform,
		},
	})
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return interactive.Message{}, NewExecutionCommandError(err.Error())
		}
		return interactive.Message{}, NewExecutionCommandError(s.Message())
	}

	if resp.Data != "" {
		return respond(resp.Data, cmdCtx), nil
	}

	if resp.Message.Type == api.BaseBodyWithFilterMessage {
		return e.filterMessage(resp.Message, cmdCtx), nil
	}

	out := interactive.Message{
		Message: resp.Message,
	}
	if !resp.Message.OnlyVisibleForYou {
		out.Description = header(cmdCtx)
	}

	return out, nil
}

func (e *PluginExecutor) Help(ctx context.Context, bindings []string, cmdCtx CommandContext) (interactive.Message, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": bindings,
		"command":  cmdCtx.CleanCmd,
	}).Debugf("Handling plugin help command...")

	cmdName := cmdCtx.Args[0]
	_, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while getting concrete plugin client: %w", err)
	}
	e.log.Debug("running help command")

	msg, err := cli.Help(ctx)
	if err != nil {
		return interactive.Message{}, err
	}

	if msg.Type == api.BaseBodyWithFilterMessage {
		return e.filterMessage(msg, cmdCtx), nil
	}

	return interactive.Message{
		Description: header(cmdCtx),
		Message:     msg,
	}, nil
}

func (e *PluginExecutor) filterMessage(msg api.Message, cmdCtx CommandContext) interactive.Message {
	// only plaintext + code block
	code := cmdCtx.ExecutorFilter.Apply(msg.BaseBody.CodeBlock)
	plaintext := cmdCtx.ExecutorFilter.Apply(msg.BaseBody.Plaintext)
	if code == "" && plaintext == "" {
		plaintext = emptyResponseMsg
	}

	outMsg := interactive.Message{
		Description: header(cmdCtx),
		Message: api.Message{
			BaseBody: api.Body{
				Plaintext: plaintext,
				CodeBlock: code,
			},
		},
	}

	// Show Filter Input if command response is more than `lineLimitToShowFilter`
	allLines := code + plaintext
	if len(strings.SplitN(allLines, "\n", lineLimitToShowFilter)) == lineLimitToShowFilter {
		outMsg.PlaintextInputs = append(outMsg.PlaintextInputs,
			filterInput(cmdCtx.CleanCmd, cmdCtx.BotName))
	}

	return outMsg
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
