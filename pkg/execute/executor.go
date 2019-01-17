package execute

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	"version":       true,
	"auth":          true,
}

var validNotifierCommands = map[string]bool{
	"notifier": true,
	"help":     true,
	"ping":     true,
}

var kubectlBinary = "/usr/local/bin/kubectl"

const (
	notifierStartMsg   = "Brace yourselves, notifications are coming."
	notifierStopMsg    = "Sure! I won't send you notifications anymore."
	unsupportedCmdMsg  = "Command not supported. Please run '@BotKube help' to see supported commands."
	kubectlDisabledMsg = "Sorry, the admin hasn't given me the permission to execute kubectl command."
)

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	Message      string
	AllowKubectl bool
}

// NewDefaultExecutor returns new Executor object
func NewDefaultExecutor(msg string, allowkubectl bool) Executor {
	return &DefaultExecutor{
		Message:      msg,
		AllowKubectl: allowkubectl,
	}
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute() string {
	args := strings.Split(e.Message, " ")
	if validKubectlCommands[args[0]] {
		if !e.AllowKubectl {
			return kubectlDisabledMsg
		}
		return runKubectlCommand(args)
	}
	if validNotifierCommands[args[0]] {
		return runNotifierCommand(args)
	}
	return unsupportedCmdMsg
}

func printHelp() string {
	allowedKubectl := ""
	for k := range validKubectlCommands {
		allowedKubectl = allowedKubectl + k + ", "
	}
	helpMsg := "BotKube executes kubectl commands on k8s cluster and returns output.\n" +
		"Usages:\n" +
		"    @BotKube <kubectl command without `kubectl` prefix>\n" +
		"e.g:\n" +
		"    @BotKube get pods\n" +
		"    @BotKube logs podname -n namespace\n" +
		"Allowed kubectl commands:\n" +
		"    " + allowedKubectl + "\n\n" +
		"Commands to manage notifier:\n" +
		"notifier stop          Stop sending k8s event notifications to Slack (started by default)\n" +
		"notifier start         Start sending k8s event notifications to Slack\n" +
		"notifier status        Show running status of event notifier\n" +
		"notifier showconfig    Show BotKube configuration for event notifier\n\n" +
		"Other Commands:\n" +
		"help                   Show help\n" +
		"ping                   Check connection health\n"
	return helpMsg

}

func printDefaultMsg() string {
	return unsupportedCmdMsg
}

func runKubectlCommand(args []string) string {
	// Use 'default' as a default namespace
	args = append([]string{"-n", "default"}, args...)

	// Remove unnecessary flags
	finalArgs := []string{}
	for _, a := range args {
		if a == "-f" || strings.HasPrefix(a, "--follow") {
			continue
		}
		if a == "-w" || strings.HasPrefix(a, "--watch") {
			continue
		}
		finalArgs = append(finalArgs, a)
	}

	cmd := exec.Command(kubectlBinary, finalArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Logger.Error("Error in executing kubectl command: ", err)
		return string(out) + err.Error()
	}
	return string(out)
}

// TODO: Have a seperate cli which runs bot commands
func runNotifierCommand(args []string) string {
	switch len(args) {
	case 1:
		if strings.ToLower(args[0]) == "help" {
			return printHelp()
		}
		if strings.ToLower(args[0]) == "ping" {
			return "pong"
		}
	case 2:
		if args[0] != "notifier" {
			return printDefaultMsg()
		}
		if args[1] == "start" {
			config.Notify = true
			log.Logger.Info("Notifier enabled")
			return notifierStartMsg
		}
		if args[1] == "stop" {
			config.Notify = false
			log.Logger.Info("Notifier disabled")
			return notifierStopMsg
		}
		if args[1] == "status" {
			if config.Notify == false {
				return "stopped"
			}
			return "running"
		}
		if args[1] == "showconfig" {
			out, err := showControllerConfig()
			if err != nil {
				log.Logger.Error("Error in executing showconfig command: ", err)
				return "Error in getting configuration!"
			}
			return out
		}
	}
	return printDefaultMsg()
}

func showControllerConfig() (string, error) {
	configPath := os.Getenv("CONFIG_PATH")
	configFile := filepath.Join(configPath, config.ConfigFileName)
	file, err := os.Open(configFile)
	defer file.Close()
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
