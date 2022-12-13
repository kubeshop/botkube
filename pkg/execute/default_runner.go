package execute

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// CommandRunner provides functionality to run arbitrary commands.
type CommandRunner interface {
	CommandCombinedOutputRunner
	CommandSeparateOutputRunner
}

// CommandCombinedOutputRunner provides functionality to run arbitrary commands.
type CommandCombinedOutputRunner interface {
	RunCombinedOutput(command string, args []string) (string, error)
}

// CommandSeparateOutputRunner provides functionality to run arbitrary commands.
type CommandSeparateOutputRunner interface {
	RunSeparateOutput(command string, args []string) (string, string, error)
}

// OSCommand provides syntax sugar for working with exec.Command
type OSCommand struct{}

// RunSeparateOutput runs a given command and returns separately its standard output and standard error.
func (*OSCommand) RunSeparateOutput(command string, args []string) (string, string, error) {
	var (
		out    strings.Builder
		outErr strings.Builder
	)

	// #nosec G204
	cmd := exec.Command(command, args...)
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	err := cmd.Run()

	return out.String(), outErr.String(), err
}

// RunCombinedOutput runs a given command and returns its combined standard output and standard error.
func (*OSCommand) RunCombinedOutput(command string, args []string) (string, error) {
	// #nosec G204
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

type (
	executorFunc    func() (interactive.Message, error)
	executorsRunner map[string]executorFunc
)

func newCmdsMapping(executors []CommandExecutor) (map[CommandVerb]map[string]CommandFn, error) {
	mappingsErrs := multierror.New()
	cmdsMapping := make(map[CommandVerb]map[string]CommandFn)
	for _, executor := range executors {
		cmds := executor.Commands()
		resNames := executor.ResourceNames()

		for verb, cmdFn := range cmds {
			if value := cmdsMapping[verb]; value == nil {
				cmdsMapping[verb] = make(map[string]CommandFn)
			}
			for _, resName := range resNames {
				if _, ok := cmdsMapping[verb][resName]; ok {
					mappingsErrs = multierror.Append(mappingsErrs, fmt.Errorf("Command collision: tried to register '%s %s', but it already exists", verb, resName))
				}
				cmdsMapping[verb][resName] = cmdFn
			}
		}
	}
	return cmdsMapping, mappingsErrs.ErrorOrNil()
}

func (cmds executorsRunner) SelectAndRun(cmdVerb string) (interactive.Message, error) {
	cmdVerb = strings.ToLower(cmdVerb)
	fn, found := cmds[cmdVerb]
	if !found {
		return interactive.Message{}, errUnsupportedCommand
	}
	return fn()
}
