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

	"github.com/kubeshop/botkube/internal/plugin"
)

// ParseCommand processes a given command string and stores the result in a given destination.
// Destination MUST be a pointer to a struct.
//
// If `-h,--help` flag was specified, arg.ErrHelp is returned and command might be not fully parsed.
//
// To support flags, positional arguments, and subcommands add dedicated `arg:` tag.
// To learn more, visit github.com/alexflint/go-arg
func ParseCommand(pluginName, command string, destination any) error {
	command = strings.TrimSpace(command)
	if !strings.HasPrefix(command, pluginName) {
		return fmt.Errorf("the input command does not target the %s plugin", pluginName)
	}
	command = strings.TrimPrefix(command, pluginName)

	p, err := arg.NewParser(arg.Config{}, destination)
	if err != nil {
		return fmt.Errorf("while creating parser: %w", err)
	}

	args, err := shellwords.Parse(command)
	if err != nil {
		return err
	}

	err = p.Parse(removeVersionFlag(args))
	if err != nil {
		return err
	}

	return nil
}

// ExecuteCommand is a simple wrapper around exec.CommandContext to simplify running a given
// command.
func ExecuteCommand(ctx context.Context, rawCmd string) (string, error) {
	return ExecuteCommandWithEnvs(ctx, rawCmd, nil)
}

// ExecuteCommandWithEnvs is a simple wrapper around exec.CommandContext to simplify running a given
// command.
func ExecuteCommandWithEnvs(ctx context.Context, rawCmd string, envs map[string]string) (string, error) {
	var stdout, stderr bytes.Buffer

	parser := shellwords.NewParser()
	parser.ParseEnv = false
	parser.ParseBacktick = false
	args, err := parser.Parse(rawCmd)
	if err != nil {
		return "", err
	}

	if len(args) < 1 {
		return "", fmt.Errorf("invalid raw command: %q", rawCmd)
	}

	bin, binArgs := args[0], args[1:]
	depDir, found := os.LookupEnv(plugin.DependencyDirEnvName)
	if found {
		// Use exactly the binary from the $PLUGIN_DEPENDENCY_DIR directory
		bin = fmt.Sprintf("%s/%s", depDir, bin)
	}

	//nolint:gosec // G204: Subprocess launched with a potential tainted input or cmd arguments
	cmd := exec.CommandContext(ctx, bin, binArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = append(cmd.Env, os.Environ()...)

	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	if err = cmd.Run(); err != nil {
		return "", runErr(stdout.String(), stderr.String(), err)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		return "", fmt.Errorf("got non-zero exit code, stdout [%q], stderr [%q]", stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func runErr(sout, serr string, err error) error {
	strBldr := strings.Builder{}
	if sout != "" {
		strBldr.WriteString(sout)
		strBldr.WriteString("\n")
	}

	if serr != "" {
		strBldr.WriteString(serr)
		strBldr.WriteString("\n")
	}

	return fmt.Errorf("%s%w", strBldr.String(), err)
}

// The go-arg library is handling the `--version` flag internally returning and error and stopping further processing, see:
// https://github.com/alexflint/go-arg/blob/727f8533acca70ca429dce4bfea729a6af75c3f7/parse.go#L610
func removeVersionFlag(args []string) []string {
	for idx := range args {
		if !strings.HasPrefix(args[idx], "--version") {
			continue
		}
		prev := idx
		next := idx + 1

		if !strings.Contains(args[idx], "=") { // val is in next arg: --version 1.2.3
			next = next + 1
		}

		if next > len(args) {
			next = len(args)
		}
		return append(args[:prev], args[next:]...)
	}
	return args
}
