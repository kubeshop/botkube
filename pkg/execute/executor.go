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
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"unicode"

	"gopkg.in/yaml.v2"

	"github.com/infracloudio/botkube/pkg/config"
	filterengine "github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

var (
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
	notifierStartMsg   = "Brace yourselves, notifications are coming from cluster '%s'."
	notifierStopMsg    = "Sure! I won't send you notifications from cluster '%s' anymore."
	unsupportedCmdMsg  = "Command not supported. Please run /botkubehelp to see supported commands."
	incompleteCmdMsg   = "You missed to pass options for the command. Please run /botkubehelp to see command options."
	kubectlDisabledMsg = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
	filterNameMissing  = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled      = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled     = "Done. I won't run '%s' filter on '%s' cluster."
)

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	Message          string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelName      string
	IsAuthChannel    bool
	DefaultNamespace string
}

// CommandRunner is an interface to run bash commands
type CommandRunner interface {
	Run() (string, error)
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

func (action FiltersAction) String() string {
	return string(action)
}

// NewDefaultExecutor returns new Executor object
func NewDefaultExecutor(msg string, allowkubectl, restrictAccess bool, defaultNamespace,
	clusterName, channelName string, isAuthChannel bool) Executor {
	return &DefaultExecutor{
		Message:          msg,
		AllowKubectl:     allowkubectl,
		RestrictAccess:   restrictAccess,
		ClusterName:      clusterName,
		ChannelName:      channelName,
		IsAuthChannel:    isAuthChannel,
		DefaultNamespace: defaultNamespace,
	}
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute() string {
	args := strings.Fields(e.Message)

	if len(args) >= 2 && utils.AllowedKubectlVerbMap[args[0]] {
		if validDebugCommands[args[0]] || // Don't check for resource if is a valid debug command
			utils.AllowedKubectlResourceMap[args[1]] || // Check if allowed resource
			utils.AllowedKubectlResourceMap[utils.KindResourceMap[strings.ToLower(args[1])]] || // Check if matches with kind name
			utils.AllowedKubectlResourceMap[utils.ShortnameResourceMap[strings.ToLower(args[1])]] { // Check if matches with short name
			isClusterNamePresent := strings.Contains(e.Message, "--cluster-name")
			if !e.AllowKubectl {
				if isClusterNamePresent && e.ClusterName == utils.GetClusterNameFromKubectlCmd(e.Message) {
					return fmt.Sprintf(kubectlDisabledMsg, e.ClusterName)
				}
				return ""
			}

			if e.RestrictAccess && !e.IsAuthChannel && isClusterNamePresent {
				return ""
			}
			return runKubectlCommand(args, e.ClusterName, e.DefaultNamespace, e.IsAuthChannel)
		}
	}
	if validNotifierCommand[args[0]] {
		return runNotifierCommand(args, e.ClusterName, e.IsAuthChannel)
	}
	if validPingCommand[args[0]] {
		res := runVersionCommand(args, e.ClusterName)
		if len(res) == 0 {
			return ""
		}
		return fmt.Sprintf("pong from cluster '%s'", e.ClusterName) + "\n\n" + res
	}
	if validVersionCommand[args[0]] {
		return runVersionCommand(args, e.ClusterName)
	}
	// Check if filter command
	if validFilterCommand[args[0]] {
		return runFilterCommand(args, e.ClusterName, e.IsAuthChannel)
	}
	if e.IsAuthChannel {
		return unsupportedCmdMsg
	}
	return ""
}

func printDefaultMsg() string {
	return unsupportedCmdMsg
}

// Trim single and double quotes from ends of string
func trimQuotes(clusterValue string) string {
	return strings.TrimFunc(clusterValue, func(r rune) bool {
		if r == unicode.SimpleFold('\u0027') || r == unicode.SimpleFold('\u0022') {
			return true
		}
		return false
	})
}

func runKubectlCommand(args []string, clusterName, defaultNamespace string, isAuthChannel bool) string {

	// run commands in namespace specified under Config.Settings.DefaultNamespace field
	if len(defaultNamespace) != 0 {
		args = append([]string{"-n", defaultNamespace}, utils.DeleteDoubleWhiteSpace(args)...)
	} else {
		args = append([]string{"-n", "default"}, utils.DeleteDoubleWhiteSpace(args)...)
	}

	// Remove unnecessary flags
	finalArgs := []string{}
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
				if index == len(args)-1 || trimQuotes(args[index+1]) != clusterName {
					return ""
				}
				isClusterNameArg = true
			} else {
				if trimQuotes(strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1]) != clusterName {
					return ""
				}
			}
			isAuthChannel = true
			continue
		}
		finalArgs = append(finalArgs, arg)
	}
	if isAuthChannel == false {
		return ""
	}
	// Get command runner
	runner := NewCommandRunner(kubectlBinary, finalArgs)
	out, err := runner.Run()
	if err != nil {
		log.Error("Error in executing kubectl command: ", err)
		return fmt.Sprintf("Cluster: %s\n%s", clusterName, out+err.Error())
	}
	return fmt.Sprintf("Cluster: %s\n%s", clusterName, out)
}

// TODO: Have a seperate cli which runs bot commands
func runNotifierCommand(args []string, clusterName string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	if len(args) < 2 {
		return incompleteCmdMsg
	}

	switch args[1] {
	case Start.String():
		config.Notify = true
		log.Info("Notifier enabled")
		return fmt.Sprintf(notifierStartMsg, clusterName)
	case Stop.String():
		config.Notify = false
		log.Info("Notifier disabled")
		return fmt.Sprintf(notifierStopMsg, clusterName)
	case Status.String():
		if config.Notify == false {
			return fmt.Sprintf("Notifications are off for cluster '%s'", clusterName)
		}
		return fmt.Sprintf("Notifications are on for cluster '%s'", clusterName)
	case ShowConfig.String():
		out, err := showControllerConfig()
		if err != nil {
			log.Error("Error in executing showconfig command: ", err)
			return "Error in getting configuration!"
		}
		return fmt.Sprintf("Showing config for cluster '%s'\n\n%s", clusterName, out)
	}
	return printDefaultMsg()
}

// runFilterCommand to list, enable or disable filters
func runFilterCommand(args []string, clusterName string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	if len(args) < 2 {
		return incompleteCmdMsg
	}

	switch args[1] {
	case FilterList.String():
		log.Debug("List filters")
		return makeFiltersList()

	// Enable filter
	case FilterEnable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, makeFiltersList())
		}
		log.Debug("Enable filters", args[2])
		if err := filterengine.DefaultFilterEngine.SetFilter(args[2], true); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterEnabled, args[2], clusterName)

	// Disable filter
	case FilterDisable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, makeFiltersList())
		}
		log.Debug("Disabled filters", args[2])
		if err := filterengine.DefaultFilterEngine.SetFilter(args[2], false); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterDisabled, args[2], clusterName)
	}
	return printDefaultMsg()
}

// Use tabwriter to display string in tabular form
// https://golang.org/pkg/text/tabwriter
func makeFiltersList() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for k, v := range filterengine.DefaultFilterEngine.ShowFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", reflect.TypeOf(k).Name(), v, k.Describe())
	}

	w.Flush()
	return buf.String()
}

func findBotKubeVersion() (versions string) {
	args := []string{"-c", fmt.Sprintf("%s version --short=true | grep Server", kubectlBinary)}
	runner := NewCommandRunner("sh", args)
	// Returns "Server Version: xxxx"
	k8sVersion, err := runner.Run()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = "Server Version: Unknown\n"
	}

	botkubeVersion := os.Getenv("BOTKUBE_VERSION")
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}
	return fmt.Sprintf("K8s %sBotKube version: %s", k8sVersion, botkubeVersion)
}

func runVersionCommand(args []string, clusterName string) string {
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
	return findBotKubeVersion()
}

func showControllerConfig() (configYaml string, err error) {
	c, err := config.New()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	// hide sensitive info
	c.Communications.Slack.Token = ""
	c.Communications.ElasticSearch.Password = ""

	b, err := yaml.Marshal(c)
	if err != nil {
		return configYaml, err
	}
	configYaml = string(b)

	return configYaml, nil
}
