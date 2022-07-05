package execute

import (
	"os/exec"
)

// DefaultCommandRunnerFunc is a wrapper for exec.Command
func DefaultCommandRunnerFunc(command string, args []string) (string, error) {
	// #nosec G204
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
