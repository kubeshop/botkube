package execute

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/alias"
	"github.com/kubeshop/botkube/pkg/execute/command"
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
	cfgManager            ConfigPersistenceManager
	commGroupName         string
	user                  string
	cmdsMapping           *CommandMapping
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute(ctx context.Context) interactive.CoreMessage {
	empty := interactive.CoreMessage{}
	rawCmd := sanitizeCommand(e.message)

	expandedRawCmd := alias.ExpandPrefix(rawCmd, e.cfg.Aliases)
	e.log.WithField("rawCmd", rawCmd).WithField("expandedRawCmd", expandedRawCmd).
		Debugf("Expanding aliases from command...")

	cmdCtx := CommandContext{
		ClusterName:     e.cfg.Settings.ClusterName,
		ExpandedRawCmd:  expandedRawCmd,
		CommGroupName:   e.commGroupName,
		User:            e.user,
		Conversation:    e.conversation,
		Platform:        e.platform,
		NotifierHandler: e.notifierHandler,
		Mapping:         e.cmdsMapping,
	}

	flags, err := ParseFlags(expandedRawCmd)
	if err != nil {
		e.log.Errorf("while parsing command flags %q: %s", expandedRawCmd, err.Error())
		return interactive.CoreMessage{
			Description: header(cmdCtx),
			Message: api.Message{
				BaseBody: api.Body{
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

	// commands below are executed only if the channel is authorized
	if !e.conversation.IsAuthenticated {
		return empty
	}

	if e.pluginExecutor.CanHandle(e.conversation.ExecutorBindings, cmdCtx.Args) {
		if isHelpCmd(cmdCtx.Args) {
			return e.ExecuteHelp(ctx, cmdCtx)
		}
		e.reportCommand(e.pluginExecutor.GetCommandPrefix(cmdCtx.Args), cmdCtx.ExecutorFilter.IsActive())
		out, err := e.pluginExecutor.Execute(ctx, e.conversation.ExecutorBindings, e.conversation.SlackState, cmdCtx)
		switch {
		case err == nil:
		case IsExecutionCommandError(err):
			return respond(err.Error(), cmdCtx)
		default:
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing command %q: %s", cmdCtx.CleanCmd, err.Error())
			return empty
		}
		return out
	}

	cmdVerb := command.Verb(strings.ToLower(cmdCtx.Args[0]))
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
		reportedCmd := string(cmdVerb)
		if cmdRes != "" {
			e.log.Infof("received unsupported resource: %q", cmdCtx.CleanCmd)
			reportedCmd = fmt.Sprintf("%s {invalid feature}", reportedCmd)
		}
		e.reportCommand(reportedCmd, false)
		msg := e.cmdsMapping.HelpMessageForVerb(cmdVerb)
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

func (e *DefaultExecutor) ExecuteHelp(ctx context.Context, cmdCtx CommandContext) interactive.CoreMessage {
	e.reportCommand(e.pluginExecutor.GetCommandPrefix(cmdCtx.Args), cmdCtx.ExecutorFilter.IsActive())
	msg, err := e.pluginExecutor.Help(ctx, e.conversation.ExecutorBindings, cmdCtx)
	if err != nil {
		e.log.Errorf("while executing help command %q: %s", cmdCtx.CleanCmd, err.Error())
		return interactive.CoreMessage{}
	}
	return msg
}

func respond(body string, cmdCtx CommandContext) interactive.CoreMessage {
	body = cmdCtx.ExecutorFilter.Apply(body)
	msgBody := api.Body{
		CodeBlock: body,
	}
	if body == "" {
		msgBody = api.Body{
			Plaintext: emptyResponseMsg,
		}
	}

	message := interactive.CoreMessage{
		Description: header(cmdCtx),
		Message: api.Message{
			BaseBody: msgBody,
		},
	}

	return appendInteractiveFilterIfNeeded(body, message, cmdCtx)
}

func sanitizeCommand(cmd string) string {
	outCmd := format.RemoveHyperlinks(cmd)
	outCmd = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(outCmd)
	outCmd = strings.TrimSpace(outCmd)
	return outCmd
}

func header(cmdCtx CommandContext) string {
	cmd := newLinePattern.ReplaceAllString(cmdCtx.ExpandedRawCmd, " ")
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

func filterInput(id string) api.LabelInput {
	return api.LabelInput{
		Command:          fmt.Sprintf("%s %s --filter=", api.MessageBotNamePlaceholder, id),
		DispatchedAction: api.DispatchInputActionOnEnter,
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

func isHelpCmd(s []string) bool {
	if len(s) < 2 {
		return false
	}
	return s[1] == "help"
}
