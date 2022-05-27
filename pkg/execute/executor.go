// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package execute

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/infracloudio/botkube/pkg/filterengine"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/pkg/version"
)

var (
	// ValidNotifierCommand is a map of valid notifier commands
	ValidNotifierCommand = map[string]bool{
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
	notifierStopMsg    = "Sure! I won't send you notifications from cluster '%s' anymore."
	unsupportedCmdMsg  = "Command not supported. Please run /botkubehelp to see supported commands."
	kubectlDisabledMsg = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
	filterNameMissing  = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled      = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled     = "Done. I won't run '%s' filter on '%s' cluster."

	// NotifierStartMsg notifier enabled response message
	NotifierStartMsg = "Brace yourselves, notifications are coming from cluster '%s'."
	// IncompleteCmdMsg incomplete command response message
	IncompleteCmdMsg = "You missed to pass options for the command. Please run /botkubehelp to see command options."
	// WrongClusterCmdMsg incomplete command response message
	WrongClusterCmdMsg = "Sorry, the admin hasn't configured me to do that for the cluster '%s'."

	// Custom messages for teams platform
	teamsUnsupportedCmdMsg = "Command not supported. Please visit botkube.io/usage to see supported commands."
)

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	cfg          config.Config
	filterEngine filterengine.FilterEngine
	log          logrus.FieldLogger
	runCmdFn     CommandRunnerFunc
	resMapping   ResourceMapping

	Message       string
	IsAuthChannel bool
	Platform      config.BotPlatform
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
	clusterName := e.cfg.Settings.ClusterName
	// Remove hyperlink if it got added automatically
	command := utils.RemoveHyperlink(e.Message)
	args := strings.Fields(strings.TrimSpace(command))
	if len(args) == 0 {
		if e.IsAuthChannel {
			return e.printDefaultMsg(e.Platform)
		}
		return "" // this prevents all bots on all clusters to answer something
	}
	if len(args) >= 1 && e.resMapping.AllowedKubectlVerbMap[args[0]] {
		if validDebugCommands[args[0]] || // Don't check for resource if is a valid debug command
			(len(args) >= 2 && (e.resMapping.AllowedKubectlResourceMap[args[1]] || // Check if allowed resource
				e.resMapping.AllowedKubectlResourceMap[e.resMapping.KindResourceMap[strings.ToLower(args[1])]] || // Check if matches with kind name
				e.resMapping.AllowedKubectlResourceMap[e.resMapping.ShortnameResourceMap[strings.ToLower(args[1])]])) { // Check if matches with short name
			isClusterNamePresent := strings.Contains(e.Message, "--cluster-name")
			allowKubectl := e.cfg.Settings.Kubectl.Enabled
			if !allowKubectl {
				if isClusterNamePresent && clusterName == utils.GetClusterNameFromKubectlCmd(e.Message) {
					return fmt.Sprintf(kubectlDisabledMsg, clusterName)
				}
				return ""
			}

			if e.cfg.Settings.Kubectl.RestrictAccess && !e.IsAuthChannel && isClusterNamePresent {
				return ""
			}
			return e.runKubectlCommand(args)
		}
	}
	if ValidNotifierCommand[args[0]] {
		return e.runNotifierCommand(args, clusterName, e.IsAuthChannel)
	}
	if validPingCommand[args[0]] {
		res := e.runVersionCommand(args, clusterName)
		if len(res) == 0 {
			return ""
		}
		return fmt.Sprintf("pong from cluster '%s'", clusterName) + "\n\n" + res
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

func (e *DefaultExecutor) printDefaultMsg(p config.BotPlatform) string {
	if p == config.TeamsBot {
		return teamsUnsupportedCmdMsg
	}
	return unsupportedCmdMsg
}

// Trim single and double quotes from ends of string
func (e *DefaultExecutor) trimQuotes(clusterValue string) string {
	return strings.TrimFunc(clusterValue, func(r rune) bool {
		if r == unicode.SimpleFold('\u0027') || r == unicode.SimpleFold('\u0022') {
			return true
		}
		return false
	})
}

func (e *DefaultExecutor) runKubectlCommand(args []string) string {
	clusterName := e.cfg.Settings.ClusterName
	defaultNamespace := e.cfg.Settings.Kubectl.DefaultNamespace
	isAuthChannel := e.IsAuthChannel
	// run commands in namespace specified under Config.Settings.DefaultNamespace field
	if !utils.Contains(args, "-n") && !utils.Contains(args, "--namespace") && len(defaultNamespace) != 0 {
		args = append([]string{"-n", defaultNamespace}, utils.DeleteDoubleWhiteSpace(args)...)
	}

	// Remove unnecessary flags
	var finalArgs []string
	isClusterNameArg := false
	for index, arg := range args {
		if isClusterNameArg {
			isClusterNameArg = false
			continue
		}
		if arg == AbbrFollowFlag.String() || strings.HasPrefix(arg, FollowFlag.String()) {
			continue
		}
		if arg == AbbrWatchFlag.String() || strings.HasPrefix(arg, WatchFlag.String()) {
			continue
		}
		// Check --cluster-name flag
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			// Check if flag value in current or next argument and compare with config.settings.clustername
			if arg == ClusterFlag.String() {
				if index == len(args)-1 || e.trimQuotes(args[index+1]) != clusterName {
					return ""
				}
				isClusterNameArg = true
			} else {
				if e.trimQuotes(strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1]) != clusterName {
					return ""
				}
			}
			isAuthChannel = true
			continue
		}
		finalArgs = append(finalArgs, arg)
	}
	if !isAuthChannel {
		return ""
	}
	// Get command runner
	out, err := e.runCmdFn(kubectlBinary, finalArgs)
	if err != nil {
		e.log.Error("Error in executing kubectl command: ", err)
		return fmt.Sprintf("Cluster: %s\n%s", clusterName, out+err.Error())
	}
	return fmt.Sprintf("Cluster: %s\n%s", clusterName, out)
}

// TODO: Have a separate cli which runs bot commands
func (e *DefaultExecutor) runNotifierCommand(args []string, clusterName string, isAuthChannel bool) string {
	if !isAuthChannel {
		return ""
	}
	if len(args) < 2 {
		return IncompleteCmdMsg
	}

	switch args[1] {
	case Start.String():
		config.Notify = true
		e.log.Info("Notifier enabled")
		return fmt.Sprintf(NotifierStartMsg, clusterName)
	case Stop.String():
		config.Notify = false
		e.log.Info("Notifier disabled")
		return fmt.Sprintf(notifierStopMsg, clusterName)
	case Status.String():
		if !config.Notify {
			return fmt.Sprintf("Notifications are off for cluster '%s'", clusterName)
		}
		return fmt.Sprintf("Notifications are on for cluster '%s'", clusterName)
	case ShowConfig.String():
		out, err := e.showControllerConfig()
		if err != nil {
			e.log.Error("Error in executing showconfig command: ", err)
			return "Error in getting configuration!"
		}
		return fmt.Sprintf("Showing config for cluster '%s'\n\n%s", clusterName, out)
	}
	return e.printDefaultMsg(e.Platform)
}

// runFilterCommand to list, enable or disable filters
func (e *DefaultExecutor) runFilterCommand(args []string, clusterName string, isAuthChannel bool) string {
	if !isAuthChannel {
		return ""
	}
	if len(args) < 2 {
		return IncompleteCmdMsg
	}

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
	return e.printDefaultMsg(e.Platform)
}

//runInfoCommand to list allowed commands
func (e *DefaultExecutor) runInfoCommand(args []string, isAuthChannel bool) string {
	if !isAuthChannel {
		return ""
	}
	if len(args) > 1 && args[1] != string(infoList) {
		return IncompleteCmdMsg
	}

	clusterName := e.cfg.Settings.ClusterName
	if len(args) > 3 && args[2] == ClusterFlag.String() && args[3] != clusterName {
		return fmt.Sprintf(WrongClusterCmdMsg, args[3])
	}

	allowedVerbs := utils.GetStringInYamlFormat("allowed verbs:", e.resMapping.AllowedKubectlVerbMap)
	allowedResources := utils.GetStringInYamlFormat("allowed resources:", e.resMapping.AllowedKubectlResourceMap)
	return allowedVerbs + allowedResources
}

// Use tabwriter to display string in tabular form
// https://golang.org/pkg/text/tabwriter
func (e *DefaultExecutor) makeFiltersList() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for k, v := range e.filterEngine.ShowFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", k.Name(), v, k.Describe())
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

const redactedSecretStr = "*** REDACTED ***"

func (e *DefaultExecutor) showControllerConfig() (string, error) {
	cfg := e.cfg

	// hide sensitive info
	// TODO: Refactor - split config into two files and avoid printing sensitive data
	// 	without need to resetting them manually (which is an error-prone approach)
	cfg.Communications.Slack.Token = redactedSecretStr
	cfg.Communications.ElasticSearch.Password = redactedSecretStr
	cfg.Communications.Discord.Token = redactedSecretStr
	cfg.Communications.Mattermost.Token = redactedSecretStr
	cfg.Communications.Lark.AppSecret = redactedSecretStr
	cfg.Communications.Lark.EncryptKey = redactedSecretStr

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
