package pluginx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/mattn/go-shellwords"
)

// ParseCommand processes a given command string and stores the result in a given destination.
// Destination MUST be a pointer to a struct.
func ParseCommand(pluginName, command string, destination any) error {
	p, err := arg.NewParser(arg.Config{}, destination)
	if err != nil {
		return fmt.Errorf("while creating parser: %w", err)
	}

	rawCmd := strings.TrimSpace(command)
	rawCmd = strings.TrimPrefix(rawCmd, pluginName)
	err = p.Parse(strings.Fields(rawCmd))
	if err != nil {
		return fmt.Errorf("while parsing input command: %w", err)
	}

	return nil
}

// ExecuteCommand is a simple wrapper around exec.CommandContext to simplify running a given
// command.
func ExecuteCommand(ctx context.Context, rawCmd string) (string, error) {
	var stdout, stderr bytes.Buffer

	envs, args, err := shellwords.ParseWithEnvs(rawCmd)
	if err != nil {
		return "", err
	}

	if len(args) < 1 {
		return "", fmt.Errorf("invalid raw command: %q", rawCmd)
	}

	if err := os.Setenv("PATH", fmt.Sprintf(`%s:%s`, pluginsDir, os.Getenv("PATH"))); err != nil {
		return "", fmt.Errorf("while updating PATH environment variable: %w", err)
	}

	//nolint:gosec // G204: Subprocess launched with a potential tainted input or cmd arguments
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, envs...)

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run command, stdout [%q], stderr [%q]: %w", stdout.String(), stderr.String(), err)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		return "", fmt.Errorf("got non-zero exit code, stdout [%q], stderr [%q]", stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}
