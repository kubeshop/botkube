package execute

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/utils"
	"github.com/kubeshop/botkube/pkg/version"
)

var (
	// validNotifierCommand is a map of valid notifier commands
	validNotifierCommand = map[string]bool{
		"notifier": true,
	}
	validPingCommand = map[string]bool{
		"ping": true,
	}
	validVersionCommand = map[string]bool{
		"version": true,
	}
	validFilterCommand = map[string]bool{
		"filters": true,
	}
	validInfoCommand = map[string]bool{
		"commands": true,
	}
	validDebugCommands = map[string]bool{
		"exec":         true,
		"logs":         true,
		"attach":       true,
		"auth":         true,
		"api-versions": true,
		"cluster-info": true,
		"cordon":       true,
		"drain":        true,
		"uncordon":     true,
	}

	kubectlBinary = "/usr/local/bin/kubectl"
)

const (
	unsupportedCmdMsg  = "Command not supported. Please run /botkubehelp to see supported commands."
	filterNameMissing  = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled      = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled     = "Done. I won't run '%s' filter on '%s' cluster."

	// incompleteCmdMsg incomplete command response message
	incompleteCmdMsg = "You missed to pass options for the command. Please run /botkubehelp to see command options."
	// WrongClusterCmdMsg incomplete command response message
	WrongClusterCmdMsg = "Sorry, the admin hasn't configured me to do that for the cluster '%s'."

	// Custom messages for teams platform
	teamsUnsupportedCmdMsg = "Command not supported. Please visit botkube.io/usage to see supported commands."

	anonymizedInvalidVerb = "{invalid verb}"
)

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	cfg              config.Config
	filterEngine     filterengine.FilterEngine
	log              logrus.FieldLogger
	runCmdFn         CommandRunnerFunc
	resMapping       ResourceMapping
	notifierExecutor *NotifierExecutor
	notifierHandler  NotifierHandler

	Message       string
	IsAuthChannel bool
	Platform      config.CommPlatformIntegration

	analyticsReporter AnalyticsReporter
	kubectlExecutor   *Kubectl
}

// CommandRunnerFunc is a function which runs arbitrary commands
type CommandRunnerFunc func(command string, args []string) (string, error)

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

// FiltersAction for options in filter commands
type FiltersAction string

// Filter command options
const (
	FilterList    FiltersAction = "list"
	FilterEnable  FiltersAction = "enable"
	FilterDisable FiltersAction = "disable"
)

// infoAction for options in Info commands
type infoAction string

// Info command options
const (
	infoList infoAction = "list"
)

func (action FiltersAction) String() string {
	return string(action)
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute() string {
	// Remove hyperlink if it got added automatically
	command := utils.RemoveHyperlink(e.Message)

	var (
		clusterName   = e.cfg.Settings.ClusterName
		inClusterName = utils.GetClusterNameFromKubectlCmd(command)
		args          = strings.Fields(strings.TrimSpace(command))
	)
	if len(args) == 0 {
		if e.IsAuthChannel {
			return e.printDefaultMsg(e.Platform)
		}
		return "" // this prevents all bots on all clusters to answer something
	}

	if inClusterName != "" && inClusterName != clusterName {
		return "" // user specified different target cluster
	}

	if e.kubectlExecutor.CanHandle(args) {
		// Currently the verb is always at the first place of `args`, and, in a result, `finalArgs`.
		// The length of the slice was already checked before
		// See the DefaultExecutor.Execute() logic.
		verb := args[0]
		err := e.analyticsReporter.ReportCommand(e.Platform, verb)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while reporting executed command: %s", err.Error())
		}
		out, err := e.kubectlExecutor.Execute(e.Message, e.IsAuthChannel)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
		}
		return out
	}
	if validNotifierCommand[args[0]] {
		if !e.IsAuthChannel {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			return ""
		}

		res, err := e.notifierExecutor.Do(args, e.Platform, clusterName, e.notifierHandler)
		if err != nil {
			if err == errInvalidNotifierCommand {
				return incompleteCmdMsg
			}

			if err == errUnsupportedCommand {
				return unsupportedCmdMsg
			}
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing notifier command: %s", err.Error())
		}

		return res
	}
	if validPingCommand[args[0]] {
		res := e.runVersionCommand(args, clusterName)
		if len(res) == 0 {
			return ""
		}
		return fmt.Sprintf("pong from cluster '%s'\n\n%s", clusterName, res)
	}
	if validVersionCommand[args[0]] {
		return e.runVersionCommand(args, clusterName)
	}
	// Check if filter command
	if validFilterCommand[args[0]] {
		return e.runFilterCommand(args, clusterName, e.IsAuthChannel)
	}

	//Check if info command
	if validInfoCommand[args[0]] {
		return e.runInfoCommand(args, e.IsAuthChannel)
	}

	if e.IsAuthChannel {
		return e.printDefaultMsg(e.Platform)
	}
	return ""
}

func (e *DefaultExecutor) printDefaultMsg(p config.CommPlatformIntegration) string {
	if p == config.TeamsCommPlatformIntegration {
		return teamsUnsupportedCmdMsg
	}
	return unsupportedCmdMsg
}

// TODO: Have a separate cli which runs bot commands
// runFilterCommand to list, enable or disable filters
func (e *DefaultExecutor) runFilterCommand(args []string, clusterName string, isAuthChannel bool) string {
	if !isAuthChannel {
		return ""
	}
	if len(args) < 2 {
		return incompleteCmdMsg
	}

	var cmdVerb = args[1]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", args[0], cmdVerb)
		err := e.analyticsReporter.ReportCommand(e.Platform, cmdToReport)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while reporting filter command: %s", err.Error())
		}
	}()

	switch args[1] {
	case FilterList.String():
		e.log.Debug("List filters")
		return e.makeFiltersList()

	// Enable filter
	case FilterEnable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList())
		}
		e.log.Debug("Enable filters", args[2])
		if err := e.filterEngine.SetFilter(args[2], true); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterEnabled, args[2], clusterName)

	// Disable filter
	case FilterDisable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList())
		}
		e.log.Debug("Disabled filters", args[2])
		if err := e.filterEngine.SetFilter(args[2], false); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterDisabled, args[2], clusterName)
	}

	cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
	return e.printDefaultMsg(e.Platform)
}

//runInfoCommand to list allowed commands
func (e *DefaultExecutor) runInfoCommand(args []string, isAuthChannel bool) string {
	if !isAuthChannel {
		return ""
	}
	if len(args) > 1 && args[1] != string(infoList) {
		return incompleteCmdMsg
	}

	err := e.analyticsReporter.ReportCommand(e.Platform, strings.Join(args, " "))
	if err != nil {
		// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
		e.log.Errorf("while reporting info command: %s", err.Error())
	}

	clusterName := e.cfg.Settings.ClusterName
	if len(args) > 3 && args[2] == ClusterFlag.String() && args[3] != clusterName {
		return fmt.Sprintf(WrongClusterCmdMsg, args[3])
	}

	allowedVerbs := e.getSortedEnabledCommands("allowed verbs", e.resMapping.AllowedKubectlVerbMap)
	allowedResources := e.getSortedEnabledCommands("allowed resources", e.resMapping.AllowedKubectlResourceMap)
	return fmt.Sprintf("%s%s", allowedVerbs, allowedResources)
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

func (e *DefaultExecutor) findBotKubeVersion() (versions string) {
	args := []string{"-c", fmt.Sprintf("%s version --short=true | grep Server", kubectlBinary)}
	// Returns "Server Version: xxxx"
	k8sVersion, err := e.runCmdFn("sh", args)
	if err != nil {
		e.log.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = "Server Version: Unknown\n"
	}

	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}
	return fmt.Sprintf("K8s %sBotKube version: %s", k8sVersion, botkubeVersion)
}

func (e *DefaultExecutor) runVersionCommand(args []string, clusterName string) string {
	err := e.analyticsReporter.ReportCommand(e.Platform, args[0])
	if err != nil {
		// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
		e.log.Errorf("while reporting version command: %s", err.Error())
	}

	checkFlag := false
	for _, arg := range args {
		if checkFlag {
			if arg != clusterName {
				return ""
			}
			checkFlag = false
			continue
		}
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			if arg == ClusterFlag.String() {
				checkFlag = true
			} else if strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1] != clusterName {
				return ""
			}
			continue
		}
	}
	return e.findBotKubeVersion()
}

func (e *DefaultExecutor) getSortedEnabledCommands(header string, commands map[string]bool) string {
	var keys []string
	for key := range commands {
		if !commands[key] {
			continue
		}

		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return fmt.Sprintf("%s: []", header)
	}

	const itemSeparator = "\n  - "
	items := strings.Join(keys, itemSeparator)
	return fmt.Sprintf("%s:%s%s\n", header, itemSeparator, items)
}
