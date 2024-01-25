package teamsxold

import (
	"fmt"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/pflag"
)

var knownTableCommands = []alphaTableCommandNames{
	{
		// TODO https://github.com/kubeshop/botkube-cloud/issues/752:
		//Command:     "list",
		//Subcommands: []string{"alias", "aliases", "als", "executor", "executors", "exec", "source", "sources", "src"},
	},
	{
		Command:     "helm",
		Subcommands: []string{"get", "list", "ls"},
	},
	{
		Command:     "kubectl",
		Subcommands: []string{"get"},
	},
}

type alphaTableCommandNames struct {
	Command     string
	Subcommands []string
}

type knownTableCommandsChecker struct {
	fullCommands     map[string]struct{}
	commandsPrefixes map[string]struct{}
}

func newKnownTableCommandsChecker() *knownTableCommandsChecker {
	commandsPrefixes := map[string]struct{}{}
	fullCommands := map[string]struct{}{}
	for _, cmd := range knownTableCommands {
		for _, subcommand := range cmd.Subcommands {
			fullCommands[fmt.Sprintf("%s %s", cmd.Command, subcommand)] = struct{}{}
		}
		commandsPrefixes[cmd.Command] = struct{}{}
	}

	return &knownTableCommandsChecker{
		fullCommands:     fullCommands,
		commandsPrefixes: commandsPrefixes,
	}
}

func (k *knownTableCommandsChecker) isKnownCommand(fullCmd string) bool {
	args, _ := shellwords.Parse(fullCmd)
	if len(args) < 2 {
		return false
	}

	cmd := fmt.Sprintf("%s %s", args[0], args[1])
	_, known := k.fullCommands[cmd]
	if !known {
		return false
	}

	if !k.isTableOutput(args) {
		return false
	}

	// TODO: for now we are not working properly with multiple tables and filter flag:
	//    kubectl get po,deploy -A --filter=argocd
	//
	// Reason: it may contains different rows with different spacing, e.g.:
	//    pod/nginx   1/1     Running   1 (27h ago)   2d5h
	//    flux-system   deployment.apps/kustomize-controller               1/1     1            1           2d5h
	//
	// In such example, current table parser will incorrectly assign values to a given columns because of different spacing in each row.
	if cmd == "kubectl get" && strings.Contains(fullCmd, ",") && strings.Contains(fullCmd, "--filter") {
		return false
	}

	return true
}

func (k *knownTableCommandsChecker) hasKnownCommandPrefix(in string) bool {
	for exp := range k.commandsPrefixes {
		if strings.Contains(in, exp) {
			return true
		}
	}
	return false
}

func (*knownTableCommandsChecker) isTableOutput(args []string) bool {
	f := pflag.NewFlagSet("extract-params", pflag.ContinueOnError)
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")
	f.ParseErrorsWhitelist.UnknownFlags = true

	var output string
	f.StringVarP(&output, "output", "o", "", "Output filter")
	if err := f.Parse(args); err != nil {
		return false
	}

	return output == "" || output == "table"
}
