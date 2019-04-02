package execute

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/infracloudio/botkube/pkg/config"
	log "github.com/infracloudio/botkube/pkg/logging"
)

var validKubectlCommands = map[string]bool{
	"api-resources": true,
	"api-versions":  true,
	"cluster-info":  true,
	"describe":      true,
	"diff":          true,
	"explain":       true,
	"get":           true,
	"logs":          true,
	"top":           true,
	"auth":          true,
}

var validNotifierCommand = map[string]bool{
	"notifier": true,
}
var validPingCommand = map[string]bool{
	"ping": true,
}
var validVersionCommand = map[string]bool{
	"version": true,
}

var kubectlBinary = "/usr/local/bin/kubectl"

const (
	notifierStartMsg   = "Brace yourselves, notifications are coming from cluster '%s'."
	notifierStopMsg    = "Sure! I won't send you notifications from cluster '%s' anymore."
	unsupportedCmdMsg  = "Command not supported. Please run /botkubehelp to see supported commands."
	kubectlDisabledMsg = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
)

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	Message       string
	AllowKubectl  bool
	ClusterName   string
	ChannelName   string
	IsAuthChannel bool
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

// NewDefaultExecutor returns new Executor object
func NewDefaultExecutor(msg string, allowkubectl bool, clusterName, channelName string, isAuthChannel bool) Executor {
	return &DefaultExecutor{
		Message:       msg,
		AllowKubectl:  allowkubectl,
		ClusterName:   clusterName,
		ChannelName:   channelName,
		IsAuthChannel: isAuthChannel,
	}
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute() string {
	args := strings.Split(e.Message, " ")
	if validKubectlCommands[args[0]] {
		if !e.AllowKubectl {
			return fmt.Sprintf(kubectlDisabledMsg, e.ClusterName)
		}
		return runKubectlCommand(args, e.ClusterName, e.IsAuthChannel)
	}
	if validNotifierCommand[args[0]] {
		return runNotifierCommand(args, e.ClusterName, e.IsAuthChannel)
	}
	if validPingCommand[args[0]] {
		res := runVersionCommand(args, e.ClusterName)
		if len(res) == 0 {
			return fmt.Sprintf("pong from cluster '%s'", e.ClusterName)
		}
		return fmt.Sprintf("pong from cluster '%s'", e.ClusterName) + "\n\n" + res
	}
	if validVersionCommand[args[0]] {
		return runVersionCommand(args, e.ClusterName)
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

func runKubectlCommand(args []string, clusterName string, isAuthChannel bool) string {
	// Use 'default' as a default namespace
	args = append([]string{"-n", "default"}, args...)

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
	cmd := exec.Command(kubectlBinary, finalArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Logger.Error("Error in executing kubectl command: ", err)
		return fmt.Sprintf("Cluster: %s\n%s", clusterName, string(out)+err.Error())
	}
	return fmt.Sprintf("Cluster: %s\n%s", clusterName, string(out))
}

// TODO: Have a seperate cli which runs bot commands
func runNotifierCommand(args []string, clusterName string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	switch args[1] {
	case Start.String():
		config.Notify = true
		log.Logger.Info("Notifier enabled")
		return fmt.Sprintf(notifierStartMsg, clusterName)
	case Stop.String():
		config.Notify = false
		log.Logger.Info("Notifier disabled")
		return fmt.Sprintf(notifierStopMsg, clusterName)
	case Status.String():
		if config.Notify == false {
			return fmt.Sprintf("Notifications are off for cluster '%s'", clusterName)
		}
		return fmt.Sprintf("Notifications are on for cluster '%s'", clusterName)
	case ShowConfig.String():
		out, err := showControllerConfig()
		if err != nil {
			log.Logger.Error("Error in executing showconfig command: ", err)
			return "Error in getting configuration!"
		}
		return fmt.Sprintf("Showing config for cluster '%s'\n\n%s", clusterName, out)
	}
	return printDefaultMsg()
}

func findBotKubeVersion() (versions string) {
	args := []string{"-c", fmt.Sprintf("%s version --short=true | grep Server", kubectlBinary)}
	cmd := exec.Command("sh", args...)
	// Returns "Server Version: xxxx"
	k8sVersion, err := cmd.CombinedOutput()
	if err != nil {
		log.Logger.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = []byte("Server Version: Unknown\n")
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
	configPath := os.Getenv("CONFIG_PATH")
	configFile := filepath.Join(configPath, config.ConfigFileName)
	file, err := os.Open(configFile)
	defer file.Close()
	if err != nil {
		return configYaml, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return configYaml, err
	}

	c := &config.Config{}
	if len(b) != 0 {
		err = yaml.Unmarshal(b, c)
		if err != nil {
			return configYaml, err
		}
	}

	// hide sensitive info
	c.Communications.Slack.Token = ""
	c.Communications.ElasticSearch.Password = ""

	b, err = yaml.Marshal(c)
	if err != nil {
		return configYaml, err
	}
	configYaml = string(b)

	return configYaml, nil
}
