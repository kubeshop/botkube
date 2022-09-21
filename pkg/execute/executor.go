package execute

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/utils"
	"github.com/kubeshop/botkube/pkg/version"
)

var (
	kubectlBinary = "/usr/local/bin/kubectl"
)

const (
	unsupportedCmdMsg = "Command not supported. Please use 'help' to see supported commands."
	filterNameMissing = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled     = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled    = "Done. I won't run '%s' filter on '%s' cluster."

	// incompleteCmdMsg incomplete command response message
	incompleteCmdMsg = "You missed to pass options for the command. Please use 'help' to see command options."

	anonymizedInvalidVerb = "{invalid verb}"
)

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	cfg               config.Config
	filterEngine      filterengine.FilterEngine
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cmdRunner         CommandSeparateOutputRunner
	notifierExecutor  *NotifierExecutor
	notifierHandler   NotifierHandler
	bindings          []string
	message           string
	isAuthChannel     bool
	platform          config.CommPlatformIntegration
	conversationID    string
	kubectlExecutor   *Kubectl
	merger            *kubectl.Merger
	cfgManager        ConfigPersistenceManager
	commGroupName     string
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
func (e *DefaultExecutor) Execute() interactive.Message {
	// Remove hyperlink if it got added automatically
	command := utils.RemoveHyperlink(e.message)

	var (
		clusterName   = e.cfg.Settings.ClusterName
		inClusterName = utils.GetClusterNameFromKubectlCmd(command)
		args          = strings.Fields(strings.TrimSpace(command))
		empty         = interactive.Message{}
	)
	if len(args) == 0 {
		if e.isAuthChannel {
			return interactive.Message{
				Base: interactive.Base{
					Description: unsupportedCmdMsg,
				},
			}
		}
		return empty // this prevents all bots on all clusters to answer something
	}

	response := func(in string) interactive.Message {
		return interactive.Message{
			Base: interactive.Base{
				Description: fmt.Sprintf("`%s` on `%s`", strings.TrimSpace(command), clusterName),
				Body: interactive.Body{
					CodeBlock: in,
				},
			},
		}
	}

	if inClusterName != "" && inClusterName != clusterName {
		e.log.WithFields(logrus.Fields{
			"config-cluster-name":  clusterName,
			"command-cluster-name": inClusterName,
		}).Debugf("Specified cluster name doesn't match ours. Ignoring further execution...")
		return empty // user specified different target cluster
	}

	if e.kubectlExecutor.CanHandle(e.bindings, args) {
		// Currently the verb is always at the first place of `args`, and, in a result, `finalArgs`.
		// The length of the slice was already checked before
		// See the DefaultExecutor.Execute() logic.
		verb := args[0]
		err := e.analyticsReporter.ReportCommand(e.platform, verb)
		if err != nil {
			e.log.Errorf("while reporting executed command: %s", err.Error())
		}
		out, err := e.kubectlExecutor.Execute(e.bindings, e.message, e.isAuthChannel)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while executing kubectl: %s", err.Error())
			return empty
		}
		return response(out)
	}

	// commands below are executed only if the channel is authorized
	if !e.isAuthChannel {
		return empty
	}

	type runner func() interactive.Message
	cmds := map[string]runner{
		"help": func() interactive.Message {
			return interactive.Help(e.platform, clusterName, e.notifierHandler.BotName())
		},
		"ping": func() interactive.Message {
			res := e.runVersionCommand("ping")
			return response(fmt.Sprintf("pong\n\n%s", res))
		},
		"version": func() interactive.Message {
			return response(e.runVersionCommand("version"))
		},
		"filters": func() interactive.Message {
			return response(e.runFilterCommand(args, clusterName))
		},
		"commands": func() interactive.Message {
			return response(e.runInfoCommand(args))
		},
		"notifier": func() interactive.Message {
			res, err := e.notifierExecutor.Do(args, e.commGroupName, e.platform, e.conversationID, clusterName, e.notifierHandler)
			switch {
			case err == nil:
			case errors.Is(err, errInvalidNotifierCommand):
				return response(incompleteCmdMsg)
			case errors.Is(err, errUnsupportedCommand):
				return response(unsupportedCmdMsg)
			default:
				// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
				e.log.Errorf("while executing notifier command: %s", err.Error())
				return empty
			}

			return response(res)
		},
		"feedback": func() interactive.Message {
			return interactive.Feedback(e.notifierHandler.BotName())
		},
	}

	handler, found := cmds[args[0]]
	if found {
		return handler()
	}

	if e.isAuthChannel {
		return response(unsupportedCmdMsg)
	}
	return empty
}

// TODO: Refactor as a part of https://github.com/kubeshop/botkube/issues/657
// runFilterCommand to list, enable or disable filters
func (e *DefaultExecutor) runFilterCommand(args []string, clusterName string) string {
	if len(args) < 2 {
		return incompleteCmdMsg
	}

	var cmdVerb = args[1]
	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", args[0], cmdVerb)
		err := e.analyticsReporter.ReportCommand(e.platform, cmdToReport)
		if err != nil {
			e.log.Errorf("while reporting filter command: %s", err.Error())
		}
	}()

	switch args[1] {
	case FilterList.String():
		e.log.Debug("List filters")
		return e.makeFiltersList()

	// Enable filter
	case FilterEnable.String():
		const enabled = true
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList())
		}
		filterName := args[2]
		e.log.Debug("Enabling filter...", filterName)
		if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
			return err.Error()
		}

		err := e.cfgManager.PersistFilterEnabled(filterName, enabled)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
		}

		return fmt.Sprintf(filterEnabled, filterName, clusterName)

	// Disable filter
	case FilterDisable.String():
		const enabled = false
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, e.makeFiltersList())
		}
		filterName := args[2]
		e.log.Debug("Disabling filter...", filterName)
		if err := e.filterEngine.SetFilter(filterName, enabled); err != nil {
			return err.Error()
		}

		err := e.cfgManager.PersistFilterEnabled(filterName, enabled)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while setting filter %q to %t: %w", filterName, enabled, err)
		}

		return fmt.Sprintf(filterDisabled, filterName, clusterName)
	}

	cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
	return unsupportedCmdMsg
}

// runInfoCommand to list allowed commands
func (e *DefaultExecutor) runInfoCommand(args []string) string {
	if len(args) > 1 && args[1] != string(infoList) {
		return incompleteCmdMsg
	}

	err := e.analyticsReporter.ReportCommand(e.platform, strings.Join(args, " "))
	if err != nil {
		e.log.Errorf("while reporting info command: %s", err.Error())
	}

	enabledKubectls, err := e.getEnabledKubectlExecutorsInChannel()
	if err != nil {
		// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
		e.log.Errorf("while rendering namespace config: %s", err.Error())
	}

	return enabledKubectls
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
func (e *DefaultExecutor) findBotKubeVersion() (versions string) {
	k8sVersion, err := e.findK8sVersion()
	if err != nil {
		e.log.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = "Unknown"
	}

	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}

	return fmt.Sprintf("K8s Server Version: %s\nBotKube version: %s", k8sVersion, botkubeVersion)
}

func (e *DefaultExecutor) runVersionCommand(cmd string) string {
	err := e.analyticsReporter.ReportCommand(e.platform, cmd)
	if err != nil {
		e.log.Errorf("while reporting version command: %s", err.Error())
	}

	return e.findBotKubeVersion()
}

func (e *DefaultExecutor) getEnabledKubectlExecutorsInChannel() (string, error) {
	type kubectlCollection map[string]config.Kubectl

	enabledKubectls := e.merger.GetAllEnabled(e.bindings)
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
