package execute

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
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

	// incompleteCmdMsg incomplete command response message
	incompleteCmdMsg = "You missed to pass options for the command. Please use 'help' to see command options."

	anonymizedInvalidVerb = "{invalid verb}"

	lineLimitToShowFilter = 16

	// TODO: build the help msg dynamically
	helpMessageList = `@Botkube list [resource]

Available resources:
actions  | action  | act          list available automations
filters  | filter  | fil          list available filters
commands | command | cmds | cmd   list enabled executors`

	helpMessageEdit = `@Botkube edit [resource]

Available resources:
sourcebindings | actsourcebinding   edit source bindings`

	helpMessageEnable = `@Botkube enable [resource]

Available resources:
actions | action | act    enable available automations
filters | filter | fil    enable available filters`

	helpMessageDisable = `@Botkube disable [resource]

Available resources:
actions | action | act    disable available automations
filters | filter | fil    disable available filters`
)

var (
	// noResourceNames is used for commands that have no resources defined
	noResourceNames       = []string{""}
	clusterNameFlagRegex  = regexp.MustCompile(`--cluster-name[=|\s]+\S+`)
	availableHelpMessages = map[CommandVerb]string{
		CommandList:    helpMessageList,
		CommandEnable:  helpMessageEnable,
		CommandDisable: helpMessageDisable,
		CommandEdit:    helpMessageEdit,
	}
)

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
	commandExecutor       *CommandsExecutor
	filterExecutor        *FilterExecutor
	pingExecutor          *PingExecutor
	versionExecutor       *VersionExecutor
	helpExecutor          *HelpExecutor
	feedbackExecutor      *FeedbackExecutor
	notifierExecutor      *NotifierExecutor
	configExecutor        *ConfigExecutor
	notifierHandler       NotifierHandler
	message               string
	platform              config.CommPlatformIntegration
	conversation          Conversation
	merger                *kubectl.Merger
	cfgManager            ConfigPersistenceManager
	commGroupName         string
	user                  string
	kubectlCmdBuilder     *KubectlCmdBuilder
	cmdsMapping           map[CommandVerb]map[string]CommandFn
}

// CommandFlags creates custom type for flags in botkube
type CommandFlags string

// Defines botkube flags
const (
	ClusterFlag    CommandFlags = "--cluster-name"
	FollowFlag     CommandFlags = "--follow"
	AbbrFollowFlag CommandFlags = "-f"
	WatchFlag      CommandFlags = "--watch"
	AbbrWatchFlag  CommandFlags = "-w"
)

func (flag CommandFlags) String() string {
	return string(flag)
}

// CommandVerb are commands supported by the bot
type CommandVerb string

// CommandVerb command options
const (
	CommandPing     CommandVerb = "ping"
	CommandHelp     CommandVerb = "help"
	CommandVersion  CommandVerb = "version"
	CommandFeedback CommandVerb = "feedback"
	CommandList     CommandVerb = "list"
	CommandEnable   CommandVerb = "enable"
	CommandDisable  CommandVerb = "disable"
	CommandEdit     CommandVerb = "edit"
	CommandStart    CommandVerb = "start"
	CommandStop     CommandVerb = "stop"
	CommandStatus   CommandVerb = "status"
	CommandConfig   CommandVerb = "config"
)

// CommandExecutor defines command structure for executors
type CommandExecutor interface {
	Commands() map[CommandVerb]CommandFn
	ResourceNames() []string
}

// CommandFn is a single command (eg. List())
type CommandFn func(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error)

// CommandContext contains the context for CommandFn
type CommandContext struct {
	Args            []string
	ClusterName     string
	CommGroupName   string
	BotName         string
	RawCmd          string
	User            string
	Conversation    Conversation
	Platform        config.CommPlatformIntegration
	ExecutorFilter  executorFilter
	NotifierHandler NotifierHandler
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute(ctx context.Context) interactive.Message {
	empty := interactive.Message{}
	rawCmd := format.RemoveHyperlinks(e.message)
	rawCmd = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(rawCmd)
	clusterName := e.cfg.Settings.ClusterName
	inClusterName := getClusterNameFromKubectlCmd(rawCmd)
	botName := e.notifierHandler.BotName()
	cmdCtx := CommandContext{
		ClusterName:     clusterName,
		CommGroupName:   e.commGroupName,
		BotName:         botName,
		RawCmd:          rawCmd,
		User:            e.user,
		Conversation:    e.conversation,
		Platform:        e.platform,
		NotifierHandler: e.notifierHandler,
	}
	execFilter, err := extractExecutorFilter(rawCmd)
	if err != nil {
		return respond(err.Error(), cmdCtx)
	}
	cmdCtx.ExecutorFilter = execFilter

	args, err := shellwords.Parse(strings.TrimSpace(execFilter.FilteredCommand()))
	if err != nil {
		e.log.Errorf("while parsing command %q: %s", execFilter.FilteredCommand(), err.Error())
		return respond("Cannot parse command. Please use 'help' to see supported commands.", cmdCtx)
	}

	if len(args) == 0 {
		if e.conversation.IsAuthenticated {
			return interactive.Message{
				Base: interactive.Base{
					Description: unsupportedCmdMsg,
				},
			}
		}
		return empty // this prevents all bots on all clusters to answer something
	}
	cmdCtx.Args = args

	if inClusterName != "" && inClusterName != clusterName {
		e.log.WithFields(logrus.Fields{
			"config-cluster-name":  clusterName,
			"command-cluster-name": inClusterName,
		}).Debugf("Specified cluster name doesn't match ours. Ignoring further execution...")
		return empty // user specified different target cluster
	}

	if e.kubectlExecutor.CanHandle(e.conversation.ExecutorBindings, args) {
		e.reportCommand(e.kubectlExecutor.GetCommandPrefix(args), execFilter.IsActive())
		out, err := e.kubectlExecutor.Execute(e.conversation.ExecutorBindings, execFilter.FilteredCommand(), e.conversation.IsAuthenticated)
		switch {
		case err == nil:
		case IsExecutionCommandError(err):
			return respond(err.Error(), cmdCtx)
		default:
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return respond(execFilter.Apply(out), cmdCtx)
	}

	// commands below are executed only if the channel is authorized
	if !e.conversation.IsAuthenticated {
		return empty
	}

	if e.kubectlCmdBuilder.CanHandle(args) {
		e.reportCommand(e.kubectlCmdBuilder.GetCommandPrefix(args), false)
		out, err := e.kubectlCmdBuilder.Do(ctx, args, e.platform, e.conversation.ExecutorBindings, e.conversation.State, botName, header(cmdCtx))
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return out
	}

	isPluginCmd := e.pluginExecutor.CanHandle(e.conversation.ExecutorBindings, args)
	if err != nil {
		// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
		e.log.Errorf("while checking if it's a plugin command: %s", err.Error())
		return empty
	}

	if isPluginCmd {
		e.reportCommand(e.pluginExecutor.GetCommandPrefix(args), execFilter.IsActive())
		out, err := e.pluginExecutor.Execute(ctx, e.conversation.ExecutorBindings, args, execFilter.FilteredCommand())
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing plugin: %s", err.Error())
			return empty
		}
		return respond(execFilter.Apply(out), cmdCtx)
	}

	cleanArgs, err := removeBotkubeRelatedFlags(args)
	if err != nil {
		e.log.Errorf("while removing Botkube related flags from arguments: %s", err.Error())
		return empty
	}

	cmdVerb := CommandVerb(strings.ToLower(cleanArgs[0]))
	var cmdRes string
	if len(cleanArgs) > 1 {
		cmdRes = strings.ToLower(cleanArgs[1])
	}

	resources, found := e.cmdsMapping[cmdVerb]
	if !found {
		e.reportCommand(anonymizedInvalidVerb, false)
		e.log.Infof("received unsupported command: %q", execFilter.FilteredCommand())
		return respond(unsupportedCmdMsg, cmdCtx)
	}

	fn, found := resources[cmdRes]
	if !found {
		e.log.Infof("received unsupported resource: %q", execFilter.FilteredCommand())
		msg := incompleteCmdMsg
		if helpMessage, ok := availableHelpMessages[cmdVerb]; ok {
			msg = helpMessage
		}
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
		e.log.Errorf("while executing command %q: %s", execFilter.FilteredCommand(), err.Error())
		msg := fmt.Sprintf(internalErrorMsgFmt, clusterName)
		return respond(msg, cmdCtx)
	}

	return msg
}

func removeBotkubeRelatedFlags(args []string) ([]string, error) {
	line := strings.Join(args, " ")
	matches := clusterNameFlagRegex.FindAllString(line, -1)

	for _, match := range matches {
		line = strings.Replace(line, match, "", 1)
	}
	return shellwords.Parse(line)
}

func respond(msg string, cmdCtx CommandContext, overrideCommand ...string) interactive.Message {
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
			Description: header(cmdCtx, overrideCommand...),
			Body:        msgBody,
		},
	}
	// Show Filter Input if command response is more than `lineLimitToShowFilter`
	if len(strings.SplitN(msg, "\n", lineLimitToShowFilter)) == lineLimitToShowFilter {
		message.PlaintextInputs = append(message.PlaintextInputs,
			filterInput(cmdCtx.ExecutorFilter.FilteredCommand(),
				cmdCtx.BotName))
	}
	return message
}

func header(cmdCtx CommandContext, overrideName ...string) string {
	cmd := fmt.Sprintf("`%s`", strings.TrimSpace(cmdCtx.RawCmd))
	if len(overrideName) > 0 {
		cmd = strings.TrimSpace(strings.Join(overrideName, " "))
	}

	out := fmt.Sprintf("%s on `%s`", cmd, cmdCtx.ClusterName)
	return appendByUserOnlyIfNeeded(out, cmdCtx.User, cmdCtx.Conversation.CommandOrigin)
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
