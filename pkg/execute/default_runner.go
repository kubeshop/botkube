package execute

import (
	"os/exec"
	"strings"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
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
	executorFunc    func() (interactive.CoreMessage, error)
	executorsRunner map[string]executorFunc
)

func (cmds executorsRunner) SelectAndRun(cmdVerb string) (interactive.CoreMessage, error) {
	cmdVerb = strings.ToLower(cmdVerb)
	fn, found := cmds[cmdVerb]
	if !found {
		return interactive.CoreMessage{}, errUnsupportedCommand
	}
	return fn()
}
