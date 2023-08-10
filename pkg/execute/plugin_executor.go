package execute

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

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
	restCfg       *rest.Config
}

// NewPluginExecutor creates a new instance of PluginExecutor.
func NewPluginExecutor(log logrus.FieldLogger, cfg config.Config, manager *plugin.Manager, restCfg *rest.Config) *PluginExecutor {
	return &PluginExecutor{
		log:           log,
		cfg:           cfg,
		pluginManager: manager,
		restCfg:       restCfg,
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

const multiWordArgPlaceholder = "{multi-word arg}"

// GetCommandPrefix returns plugin name and the verb if present. If the verb is multi-word, it returns a placeholder for it.
func (e *PluginExecutor) GetCommandPrefix(args []string) string {
	switch len(args) {
	case 0:
		return ""
	case 1:
		return args[0]
	}

	if strings.Contains(args[1], " ") {
		return fmt.Sprintf("%s %s", args[0], multiWordArgPlaceholder)
	}

	return fmt.Sprintf("%s %s", args[0], args[1])
}

// Execute executes plugin executor based on a given command.
func (e *PluginExecutor) Execute(ctx context.Context, bindings []string, slackState *slack.BlockActionStates, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": bindings,
		"command":  cmdCtx.CleanCmd,
	}).Debug("Handling plugin command...")

	cmdName := cmdCtx.Args[0]
	plugins, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	configs, err := e.collectConfigs(plugins)
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while collecting configs: %w", err)
	}

	input := plugin.KubeConfigInput{
		Channel: cmdCtx.Conversation.DisplayName,
	}
	kubeconfig, err := plugin.GenerateKubeConfig(e.restCfg, e.cfg.Settings.ClusterName, plugins[0].Context, input)
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while generating kube config: %w", err)
	}

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while getting concrete plugin client: %w", err)
	}

	if slackState != nil {
		e.sanitizeSlackStateIDs(slackState)
	}

	resp, err := cli.Execute(ctx, executor.ExecuteInput{
		Command: cmdCtx.CleanCmd,
		Configs: configs,
		Context: executor.ExecuteInputContext{
			IsInteractivitySupported: cmdCtx.Platform.IsInteractive(),
			SlackState:               slackState,
			KubeConfig:               kubeconfig,
		},
	})
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return interactive.CoreMessage{}, NewExecutionCommandError(err.Error())
		}
		return interactive.CoreMessage{}, NewExecutionCommandError(s.Message())
	}

	if resp.Data != "" {
		return respond(resp.Data, cmdCtx), nil
	}

	if resp.Message.IsEmpty() {
		return emptyMsg(cmdCtx), nil
	}

	if resp.Message.Type == api.BaseBodyWithFilterMessage {
		return e.filterMessage(resp.Message, cmdCtx), nil
	}

	out := interactive.CoreMessage{
		Message: resp.Message,
	}
	if !resp.Message.OnlyVisibleForYou {
		out.Description = header(cmdCtx)
	}

	return out, nil
}

// sanitizeSlackStateIDs makes sure that the slack state doesn't contain the --cluster-name
// in the ids. Example state before sanitizing:
//
//	Values: map[string]map[string]slack.BlockAction{
//	  "6f79d399-e607-4744-80e8-6eb8e5a7743e": map[string]slack.BlockAction{
//	    "kubectl @builder --resource-type --cluster-name=`"stage\"": slack.BlockAction{}
//	  }
//	}
func (e *PluginExecutor) sanitizeSlackStateIDs(slackState *slack.BlockActionStates) {
	flag := fmt.Sprintf("--cluster-name=%q ", e.cfg.Settings.ClusterName)

	for id, blocks := range slackState.Values {
		updated := make(map[string]slack.BlockAction, len(blocks))
		for id, act := range blocks {
			cleanID := strings.ReplaceAll(id, flag, "")
			act.Value = strings.ReplaceAll(act.Value, flag, "")
			act.SelectedOption.Value = strings.ReplaceAll(act.SelectedOption.Value, flag, "")
			act.ActionID = strings.ReplaceAll(act.ActionID, flag, "")
			act.BlockID = strings.ReplaceAll(act.BlockID, flag, "")
			updated[cleanID] = act
		}

		delete(slackState.Values, id)
		cleanID := strings.ReplaceAll(id, flag, "")
		slackState.Values[cleanID] = updated
	}
}

func (e *PluginExecutor) Help(ctx context.Context, bindings []string, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	e.log.WithFields(logrus.Fields{
		"bindings": bindings,
		"command":  cmdCtx.CleanCmd,
	}).Debugf("Handling plugin help command...")

	cmdName := cmdCtx.Args[0]
	_, fullPluginName := e.getEnabledPlugins(bindings, cmdName)

	cli, err := e.pluginManager.GetExecutor(fullPluginName)
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while getting concrete plugin client: %w", err)
	}
	e.log.Debug("running help command")

	msg, err := cli.Help(ctx)
	if err != nil {
		return interactive.CoreMessage{}, err
	}

	if msg.IsEmpty() {
		return emptyMsg(cmdCtx), nil
	}

	if msg.Type == api.BaseBodyWithFilterMessage {
		return e.filterMessage(msg, cmdCtx), nil
	}

	return interactive.CoreMessage{
		Description: header(cmdCtx),
		Message:     msg,
	}, nil
}

func emptyMsg(cmdCtx CommandContext) interactive.CoreMessage {
	return interactive.CoreMessage{
		Description: header(cmdCtx),
		Message: api.Message{
			BaseBody: api.Body{
				Plaintext: emptyResponseMsg,
			},
		},
	}
}

// filterMessage takes into account only base plaintext + code block, all other properties are ignored.
// This method should be called only for message type api.BaseBodyWithFilterMessage.
func (e *PluginExecutor) filterMessage(msg api.Message, cmdCtx CommandContext) interactive.CoreMessage {
	code := cmdCtx.ExecutorFilter.Apply(msg.BaseBody.CodeBlock)
	plaintext := cmdCtx.ExecutorFilter.Apply(msg.BaseBody.Plaintext)
	if code == "" && plaintext == "" {
		plaintext = emptyResponseMsg
	}

	outMsg := interactive.CoreMessage{
		Description: header(cmdCtx),
		Message: api.Message{
			BaseBody: api.Body{
				Plaintext: plaintext,
				CodeBlock: code,
			},
		},
	}

	allLines := code + plaintext
	return appendInteractiveFilterIfNeeded(allLines, outMsg, cmdCtx)
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
