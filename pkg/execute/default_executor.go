// +build !test

package execute

import (
	"os/exec"
)

// DefaultRunner contains default implementation for Run
type DefaultRunner struct {
	command string
	args    []string
}

// NewCommandRunner returns new DefaultRunner
func NewCommandRunner(command string, args []string) CommandRunner {
	return DefaultRunner{
		command: command,
		args:    args,
	}
}

// Run executes bash command
func (r DefaultRunner) Run() (string, error) {
	cmd := exec.Command(r.command, r.args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
