package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/format"
	"github.com/kubeshop/botkube/pkg/version"
)

var (
	kubectlBinary = "/usr/local/bin/kubectl"
)

const (
	unsupportedCmdMsg   = "Command not supported. Please use 'help' to see supported commands."
	filterNameMissing   = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	actionNameMissing   = "You forgot to pass action name. Please pass one of the following valid actions:\n\n%s"
	actionEnabled       = "I have enabled '%s' action on '%s' cluster."
	actionDisabled      = "Done. I won't run '%s' action on '%s' cluster."
	filterEnabled       = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled      = "Done. I won't run '%s' filter on '%s' cluster."
	internalErrorMsgFmt = "Sorry, an internal error occurred while executing your command for the '%s' cluster :( See the logs for more details."
	emptyResponseMsg    = ".... empty response _*<cricket sounds>*_ :cricket: :cricket: :cricket:"

	// incompleteCmdMsg incomplete command response message
	incompleteCmdMsg = "You missed to pass options for the command. Please use 'help' to see command options."

	anonymizedInvalidVerb = "{invalid verb}"

	// Currently we support only `kubectl, so we
	// override the message to human-readable command name.
	humanReadableCommandListName = "Available kubectl commands"

	lineLimitToShowFilter = 16
)

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	cfg               config.Config
	filterEngine      filterengine.FilterEngine
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cmdRunner         CommandSeparateOutputRunner
	kubectlExecutor   *Kubectl
	editExecutor      *EditExecutor
	actionExecutor    *ActionExecutor
	notifierExecutor  *NotifierExecutor
	notifierHandler   NotifierHandler
	message           string
	platform          config.CommPlatformIntegration
	conversation      Conversation
	merger            *kubectl.Merger
	cfgManager        ConfigPersistenceManager
	commGroupName     string
	user              string
	kubectlCmdBuilder *KubectlCmdBuilder
}

// NotifierAction creates custom type for notifier actions
type NotifierAction string

// Defines constants for notifier actions
const (
	Start      NotifierAction = "start"
	Stop       NotifierAction = "stop"
	Status     NotifierAction = "status"
	ShowConfig NotifierAction = "showconfig"
)

func (action NotifierAction) String() string {
	return string(action)
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
	CommandList    CommandVerb = "list"
	CommandEnable  CommandVerb = "enable"
	CommandDisable CommandVerb = "disable"
)

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute(ctx context.Context) interactive.Message {
	empty := interactive.Message{}
	rawCmd := format.RemoveHyperlinks(e.message)
	rawCmd = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(rawCmd)
	clusterName := e.cfg.Settings.ClusterName
	inClusterName := getClusterNameFromKubectlCmd(rawCmd)
	botName := e.notifierHandler.BotName()

	execFilter, err := extractExecutorFilter(rawCmd)
	if err != nil {
		return e.respond(err.Error(), rawCmd, "", botName)
	}

	args := strings.Fields(rawCmd)
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
			return e.respond(err.Error(), rawCmd, execFilter.FilteredCommand(), botName)
		default:
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return e.respond(execFilter.Apply(out), rawCmd, execFilter.FilteredCommand(), botName)
	}

	// commands below are executed only if the channel is authorized
	if !e.conversation.IsAuthenticated {
		return empty
	}

	if e.kubectlCmdBuilder.CanHandle(args) {
		e.reportCommand(e.kubectlCmdBuilder.GetCommandPrefix(args), false)
		out, err := e.kubectlCmdBuilder.Do(ctx, args, e.platform, e.conversation.ExecutorBindings, e.conversation.State, botName, e.header(rawCmd))
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return out
	}

	cmds := executorsRunner{
		"help": func() (interactive.Message, error) {
			e.reportCommand(args[0], false)
			return interactive.NewHelpMessage(e.platform, clusterName, botName).Build(), nil
		},
		"ping": func() (interactive.Message, error) {
			res := e.runVersionCommand("ping")
			return e.respond(fmt.Sprintf("pong\n\n%s", res), rawCmd, execFilter.FilteredCommand(), botName), nil
		},
		"version": func() (interactive.Message, error) {
			return e.respond(e.runVersionCommand("version"), rawCmd, execFilter.FilteredCommand(), botName), nil
		},
		"filters": func() (interactive.Message, error) {
			res, err := e.runFilterCommand(ctx, args, clusterName)
			return e.respond(execFilter.Apply(res), rawCmd, execFilter.FilteredCommand(), botName), err
		},
		"commands": func() (interactive.Message, error) {
			res, err := e.runInfoCommand(args, execFilter.IsActive())
			return e.respond(execFilter.Apply(res), rawCmd, execFilter.FilteredCommand(), botName, humanReadableCommandListName), err
		},
		"notifier": func() (interactive.Message, error) {
			res, err := e.notifierExecutor.Do(ctx, args, e.commGroupName, e.platform, e.conversation, clusterName, e.notifierHandler)
			return e.respond(res, rawCmd, execFilter.FilteredCommand(), botName), err
		},
		"edit": func() (interactive.Message, error) {
			return e.editExecutor.Do(args, e.commGroupName, e.platform, e.conversation, e.user, botName)
		},
		"feedback": func() (interactive.Message, error) {
			e.reportCommand(args[0], false)
			return interactive.Feedback(), nil
		},
		"list": func() (interactive.Message, error) {
			res, err := e.actionExecutor.Do(ctx, args, clusterName, e.conversation, e.platform)
			return e.respond(res, rawCmd, execFilter.FilteredCommand(), botName), err
		},
		"enable": func() (interactive.Message, error) {
			res, err := e.actionExecutor.Do(ctx, args, clusterName, e.conversation, e.platform)
			return e.respond(res, rawCmd, execFilter.FilteredCommand(), botName), err
		},
		"disable": func() (interactive.Message, error) {
			res, err := e.actionExecutor.Do(ctx, args, clusterName, e.conversation, e.platform)
			return e.respond(res, rawCmd, execFilter.FilteredCommand(), botName), err
		},
	}

	msg, err := cmds.SelectAndRun(args[0])
	switch {
	case err == nil:
	case errors.Is(err, errInvalidCommand):
		return e.respond(incompleteCmdMsg, rawCmd, execFilter.FilteredCommand(), botName)
	case errors.Is(err, errUnsupportedCommand):
		return e.respond(unsupportedCmdMsg, rawCmd, execFilter.FilteredCommand(), botName)
	case IsExecutionCommandError(err):
		return e.respond(err.Error(), rawCmd, execFilter.FilteredCommand(), botName)
	default:
		e.log.Errorf("while executing command %q: %s", execFilter.FilteredCommand(), err.Error())
		internalErrorMsg := fmt.Sprintf(internalErrorMsgFmt, clusterName)
		return e.respond(internalErrorMsg, rawCmd, execFilter.FilteredCommand(), botName)
	}

	return msg
}

func (e *DefaultExecutor) respond(msg string, rawCmd string, filteredCmd string, botName string, overrideCommand ...string) interactive.Message {
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
			Description: e.header(rawCmd, overrideCommand...),
			Body:        msgBody,
		},
	}
	// Show Filter Input if command response is more than `lineLimitToShowFilter`
	if len(strings.SplitN(msg, "\n", lineLimitToShowFilter)) == lineLimitToShowFilter {
		message.PlaintextInputs = append(message.PlaintextInputs, e.filterInput(filteredCmd, botName))
	}
	return message
}

func (e *DefaultExecutor) header(command string, overrideName ...string) string {
	cmd := fmt.Sprintf("`%s`", strings.TrimSpace(command))
	if len(overrideName) > 0 {
		cmd = strings.TrimSpace(strings.Join(overrideName, " "))
	}

	out := fmt.Sprintf("%s on `%s`", cmd, e.cfg.Settings.ClusterName)
	return e.appendByUserOnlyIfNeeded(out)
}

func (e *DefaultExecutor) reportCommand(verb string, withFilter bool) {
	err := e.analyticsReporter.ReportCommand(e.platform, verb, e.conversation.CommandOrigin, withFilter)
	if err != nil {
		e.log.Errorf("while reporting %s command: %s", verb, err.Error())
	}
}

// TODO: Refactor as a part of https://github.com/kubeshop/botkube/issues/657
// runFilterCommand to list, enable or disable filters
func (e *DefaultExecutor) runFilterCommand(ctx context.Context, args []string, clusterName string) (string, error) {
	if len(args) < 2 {
		return "", errInvalidCommand
	}

	var cmdVerb = args[1]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", args[0], cmdVerb)
		e.reportCommand(cmdToReport, false)
	}()

	switch CommandVerb(args[1]) {
	case CommandList:
		e.log.Debug("List filters")
		return e.makeFiltersList(), nil

	// Enable filter
	case CommandEnable:
		const enabled = true
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList()), nil
		}
		filterName := args[2]
		e.log.Debug("Enabling filter...", filterName)
		if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
			return err.Error(), nil
		}

		err := e.cfgManager.PersistFilterEnabled(ctx, filterName, enabled)
		if err != nil {
			return "", fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
		}

		return fmt.Sprintf(filterEnabled, filterName, clusterName), nil

	// Disable filter
	case CommandDisable:
		const enabled = false
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList()), nil
		}
		filterName := args[2]
		e.log.Debug("Disabling filter...", filterName)
		if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
			return err.Error(), nil
		}

		err := e.cfgManager.PersistFilterEnabled(ctx, filterName, enabled)
		if err != nil {
			return "", fmt.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
		}

		return fmt.Sprintf(filterDisabled, filterName, clusterName), nil
	}

	cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
	return "", errUnsupportedCommand
}

// runInfoCommand to list allowed commands
func (e *DefaultExecutor) runInfoCommand(args []string, withFilter bool) (string, error) {
	if len(args) < 2 {
		return "", errInvalidCommand
	}
	var cmdVerb = args[1]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", args[0], cmdVerb)
		e.reportCommand(cmdToReport, withFilter)
	}()

	switch CommandVerb(cmdVerb) {
	case CommandList:
		enabledKubectls, err := e.getEnabledKubectlExecutorsInChannel()
		if err != nil {
			return "", fmt.Errorf("while rendering namespace config: %s", err.Error())
		}

		return enabledKubectls, nil
	}

	cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
	return "", errUnsupportedCommand
}

// Use tabwriter to display string in tabular form
// https://golang.org/pkg/text/tabwriter
func (e *DefaultExecutor) makeFiltersList() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for _, filter := range e.filterEngine.RegisteredFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", filter.Name(), filter.Enabled, filter.Describe())
	}

	w.Flush()
	return buf.String()
}

type kubectlVersionOutput struct {
	Server struct {
		GitVersion string `json:"gitVersion"`
	} `json:"serverVersion"`
}

func (e *DefaultExecutor) findK8sVersion() (string, error) {
	args := []string{"-c", fmt.Sprintf("%s version --output=json", kubectlBinary)}
	stdout, stderr, err := e.cmdRunner.RunSeparateOutput("sh", args)
	e.log.Debugf("Raw kubectl version output: %q", stdout)
	if err != nil {
		return "", fmt.Errorf("unable to execute kubectl version: %w [%q]", err, stderr)
	}

	var out kubectlVersionOutput
	err = json.Unmarshal([]byte(stdout), &out)
	if err != nil {
		return "", err
	}
	if out.Server.GitVersion == "" {
		return "", fmt.Errorf("unable to unmarshal server git version from %q", stdout)
	}

	ver := out.Server.GitVersion
	if stderr != "" {
		ver += "\n" + stderr
	}

	return ver, nil
}
func (e *DefaultExecutor) findBotkubeVersion() (versions string) {
	k8sVersion, err := e.findK8sVersion()
	if err != nil {
		e.log.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = "Unknown"
	}

	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}

	return fmt.Sprintf("K8s Server Version: %s\nBotkube version: %s", k8sVersion, botkubeVersion)
}

func (e *DefaultExecutor) runVersionCommand(cmd string) string {
	err := e.analyticsReporter.ReportCommand(e.platform, cmd, e.conversation.CommandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting version command: %s", err.Error())
	}

	return e.findBotkubeVersion()
}

func (e *DefaultExecutor) getEnabledKubectlExecutorsInChannel() (string, error) {
	type kubectlCollection map[string]config.Kubectl

	enabledKubectls := e.merger.GetAllEnabled(e.conversation.ExecutorBindings)
	out := map[string]map[string]kubectlCollection{
		"Enabled executors": {
			"kubectl": enabledKubectls,
		},
	}

	var buff strings.Builder
	encode := yaml.NewEncoder(&buff)
	encode.SetIndent(2)
	err := encode.Encode(out)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}

// appendByUserOnlyIfNeeded returns the "by Foo" only if the command was executed via button.
func (e *DefaultExecutor) appendByUserOnlyIfNeeded(cmd string) string {
	if e.user == "" || e.conversation.CommandOrigin == command.TypedOrigin {
		return cmd
	}
	return fmt.Sprintf("%s by %s", cmd, e.user)
}

func (e *DefaultExecutor) filterInput(id, botName string) interactive.LabelInput {
	return interactive.LabelInput{
		Command:          fmt.Sprintf("%s %s --filter=", botName, id),
		DispatchedAction: interactive.DispatchInputActionOnEnter,
		Placeholder:      "String pattern to filter by",
		Text:             "Filter output",
	}
}
