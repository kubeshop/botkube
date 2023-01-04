package execute

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/format"
)

const (
	unsupportedCmdMsg   = "Command not supported. Please use 'help' to see supported commands."
	internalErrorMsgFmt = "Sorry, an internal error occurred while executing your command for the '%s' cluster :( See the logs for more details."
	emptyResponseMsg    = ".... empty response _*<cricket sounds>*_ :cricket: :cricket: :cricket:"

	anonymizedInvalidVerb = "{invalid verb}"

	lineLimitToShowFilter = 16
)

var newLinePattern = regexp.MustCompile(`\r?\n`)

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	cfg                   config.Config
	filterEngine          filterengine.FilterEngine
	log                   logrus.FieldLogger
	analyticsReporter     AnalyticsReporter
	kubectlExecutor       *Kubectl
	pluginExecutor        *PluginExecutor
	sourceBindingExecutor *SourceBindingExecutor
	actionExecutor        *ActionExecutor
	filterExecutor        *FilterExecutor
	pingExecutor          *PingExecutor
	versionExecutor       *VersionExecutor
	helpExecutor          *HelpExecutor
	feedbackExecutor      *FeedbackExecutor
	notifierExecutor      *NotifierExecutor
	configExecutor        *ConfigExecutor
	execExecutor          *ExecExecutor
	sourceExecutor        *SourceExecutor
	notifierHandler       NotifierHandler
	message               string
	platform              config.CommPlatformIntegration
	conversation          Conversation
	merger                *kubectl.Merger
	cfgManager            ConfigPersistenceManager
	commGroupName         string
	user                  string
	kubectlCmdBuilder     *KubectlCmdBuilder
	cmdsMapping           *CommandMapping
}

// CommandFlags creates custom type for flags in botkube
type CommandFlags string

// Defines botkube flags
const (
	FollowFlag     CommandFlags = "--follow"
	AbbrFollowFlag CommandFlags = "-f"
	WatchFlag      CommandFlags = "--watch"
	AbbrWatchFlag  CommandFlags = "-w"
)

func (flag CommandFlags) String() string {
	return string(flag)
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute(ctx context.Context) interactive.Message {
	empty := interactive.Message{}
	rawCmd := format.RemoveHyperlinks(e.message)
	rawCmd = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(rawCmd)
	cmdCtx := CommandContext{
		ClusterName:     e.cfg.Settings.ClusterName,
		RawCmd:          rawCmd,
		CommGroupName:   e.commGroupName,
		BotName:         e.notifierHandler.BotName(),
		User:            e.user,
		Conversation:    e.conversation,
		Platform:        e.platform,
		NotifierHandler: e.notifierHandler,
		Mapping:         e.cmdsMapping,
	}

	flags, err := ParseFlags(rawCmd)
	if err != nil {
		e.log.Errorf("while parsing command flags %q: %s", rawCmd, err.Error())
		return interactive.Message{
			Base: interactive.Base{
				Description: header(cmdCtx),
				Body: interactive.Body{
					Plaintext: err.Error(),
				},
			},
		}
	}

	cmdCtx.CleanCmd = flags.CleanCmd
	cmdCtx.ProvidedClusterName = flags.ClusterName
	cmdCtx.Args = flags.TokenizedCmd
	cmdCtx.ExecutorFilter = newExecutorTextFilter(flags.Filter)

	if len(cmdCtx.Args) == 0 {
		if e.conversation.IsAuthenticated {
			msg, err := e.helpExecutor.Help(ctx, cmdCtx)
			if err != nil {
				e.log.Errorf("while getting help message: %s", err.Error())
				return respond(err.Error(), cmdCtx)
			}
			return msg
		}
		return empty // this prevents all bots on all clusters to answer something
	}

	if !cmdCtx.ProvidedClusterNameEqualOrEmpty() {
		e.log.WithFields(logrus.Fields{
			"config-cluster-name":  cmdCtx.ClusterName,
			"command-cluster-name": cmdCtx.ProvidedClusterName,
		}).Debugf("Specified cluster name doesn't match ours. Ignoring further execution...")
		return empty // user specified different target cluster
	}

	if e.kubectlExecutor.CanHandle(e.conversation.ExecutorBindings, cmdCtx.Args) {
		e.reportCommand(e.kubectlExecutor.GetCommandPrefix(cmdCtx.Args), cmdCtx.ExecutorFilter.IsActive())
		out, err := e.kubectlExecutor.Execute(e.conversation.ExecutorBindings, cmdCtx.CleanCmd, e.conversation.IsAuthenticated, cmdCtx)
		switch {
		case err == nil:
		case IsExecutionCommandError(err):
			return respond(err.Error(), cmdCtx)
		default:
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return respond(out, cmdCtx)
	}

	// commands below are executed only if the channel is authorized
	if !e.conversation.IsAuthenticated {
		return empty
	}

	if e.kubectlCmdBuilder.CanHandle(cmdCtx.Args) {
		e.reportCommand(e.kubectlCmdBuilder.GetCommandPrefix(cmdCtx.Args), false)
		out, err := e.kubectlCmdBuilder.Do(ctx, cmdCtx.Args, e.platform, e.conversation.ExecutorBindings, e.conversation.State, cmdCtx.BotName, header(cmdCtx), cmdCtx)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return out
	}

	isPluginCmd := e.pluginExecutor.CanHandle(e.conversation.ExecutorBindings, cmdCtx.Args)
	if err != nil {
		// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
		e.log.Errorf("while checking if it's a plugin command: %s", err.Error())
		return empty
	}

	if isPluginCmd {
		e.reportCommand(e.pluginExecutor.GetCommandPrefix(cmdCtx.Args), cmdCtx.ExecutorFilter.IsActive())
		out, err := e.pluginExecutor.Execute(ctx, e.conversation.ExecutorBindings, cmdCtx.Args, cmdCtx.CleanCmd)
		switch {
		case err == nil:
		case IsExecutionCommandError(err):
			return respond(err.Error(), cmdCtx)
		default:
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing command %q: %s", cmdCtx.CleanCmd, err.Error())
			return empty
		}
		return respond(out, cmdCtx)
	}

	cmdVerb := CommandVerb(strings.ToLower(cmdCtx.Args[0]))
	var cmdRes string
	if len(cmdCtx.Args) > 1 {
		cmdRes = strings.ToLower(cmdCtx.Args[1])
	}

	fn, foundRes, foundFn := e.cmdsMapping.FindFn(cmdVerb, cmdRes)
	if !foundRes {
		e.reportCommand(anonymizedInvalidVerb, false)
		e.log.Infof("received unsupported command: %q", cmdCtx.CleanCmd)
		return respond(unsupportedCmdMsg, cmdCtx)
	}

	if !foundFn {
		e.reportCommand(fmt.Sprintf("%s {invalid feature}", cmdVerb), false)
		e.log.Infof("received unsupported resource: %q", cmdCtx.CleanCmd)
		msg := e.cmdsMapping.HelpMessageForVerb(cmdVerb, cmdCtx.BotName)
		return respond(msg, cmdCtx)
	}

	msg, err := fn(ctx, cmdCtx)
	switch {
	case err == nil:
	case errors.Is(err, errInvalidCommand):
		return respond(incompleteCmdMsg, cmdCtx)
	case errors.Is(err, errUnsupportedCommand):
		return respond(unsupportedCmdMsg, cmdCtx)
	case IsExecutionCommandError(err):
		return respond(err.Error(), cmdCtx)
	default:
		e.log.Errorf("while executing command %q: %s", cmdCtx.CleanCmd, err.Error())
		msg := fmt.Sprintf(internalErrorMsgFmt, cmdCtx.ClusterName)
		return respond(msg, cmdCtx)
	}

	return msg
}

func respond(msg string, cmdCtx CommandContext) interactive.Message {
	msg = cmdCtx.ExecutorFilter.Apply(msg)
	msgBody := interactive.Body{
		CodeBlock: msg,
	}
	if msg == "" {
		msgBody = interactive.Body{
			Plaintext: emptyResponseMsg,
		}
	}

	message := interactive.Message{
		Base: interactive.Base{
			Description: header(cmdCtx),
			Body:        msgBody,
		},
	}
	// Show Filter Input if command response is more than `lineLimitToShowFilter`
	if len(strings.SplitN(msg, "\n", lineLimitToShowFilter)) == lineLimitToShowFilter {
		message.PlaintextInputs = append(message.PlaintextInputs,
			filterInput(cmdCtx.CleanCmd,
				cmdCtx.BotName))
	}
	return message
}

func header(cmdCtx CommandContext) string {
	cmd := newLinePattern.ReplaceAllString(cmdCtx.RawCmd, " ")
	cmd = removeMultipleSpaces(cmd)
	cmd = strings.TrimSpace(cmd)
	cmd = fmt.Sprintf("`%s`", cmd)

	out := fmt.Sprintf("%s on `%s`", cmd, cmdCtx.ClusterName)
	return appendByUserOnlyIfNeeded(out, cmdCtx.User, cmdCtx.Conversation.CommandOrigin)
}

func removeMultipleSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (e *DefaultExecutor) reportCommand(verb string, withFilter bool) {
	err := e.analyticsReporter.ReportCommand(e.platform, verb, e.conversation.CommandOrigin, withFilter)
	if err != nil {
		e.log.Errorf("while reporting %s command: %s", verb, err.Error())
	}
}

// appendByUserOnlyIfNeeded returns the "by Foo" only if the command was executed via button.
func appendByUserOnlyIfNeeded(cmd, user string, origin command.Origin) string {
	if user == "" || origin == command.TypedOrigin {
		return cmd
	}
	return fmt.Sprintf("%s by %s", cmd, user)
}

func filterInput(id, botName string) interactive.LabelInput {
	return interactive.LabelInput{
		Command:          fmt.Sprintf("%s %s --filter=", botName, id),
		DispatchedAction: interactive.DispatchInputActionOnEnter,
		Placeholder:      "String pattern to filter by",
		Text:             "Filter output",
	}
}

func parseCmdVerb(args []string) (cmd, verb string) {
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
	}
	if len(args) > 1 {
		verb = strings.ToLower(args[1])
	}
	return
}
