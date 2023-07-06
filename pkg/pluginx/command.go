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

// ExecuteCommandOutput holds ExecuteCommand output.
type ExecuteCommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecuteCommandWithEnvs is a simple wrapper around exec.CommandContext to simplify running a given
// command.
//
// Deprecated: Use ExecuteCommand(ctx, rawCmd, ExecuteCommandEnvs(envs)) instead.
func ExecuteCommandWithEnvs(ctx context.Context, rawCmd string, envs map[string]string) (string, error) {
	out, err := ExecuteCommand(ctx, rawCmd, ExecuteCommandEnvs(envs))
	if err != nil {
		return "", err
	}
	return out.Stdout, nil
}

// ExecuteCommand is a simple wrapper around exec.CommandContext to simplify running a given command.
func ExecuteCommand(ctx context.Context, rawCmd string, mutators ...ExecuteCommandMutation) (ExecuteCommandOutput, error) {
	opts := ExecuteCommandOptions{
		DependencyDir: os.Getenv(plugin.DependencyDirEnvName),
	}
	for _, mutate := range mutators {
		mutate(&opts)
	}

	var stdout, stderr bytes.Buffer

	parser := shellwords.NewParser()
	parser.ParseEnv = false
	parser.ParseBacktick = false
	args, err := parser.Parse(rawCmd)
	if err != nil {
		return ExecuteCommandOutput{}, err
	}

	if len(args) < 1 {
		return ExecuteCommandOutput{}, fmt.Errorf("invalid raw command: %q", rawCmd)
	}

	bin, binArgs := args[0], args[1:]
	if opts.DependencyDir != "" {
		// Use exactly the binary from the dependency  directory
		bin = fmt.Sprintf("%s/%s", opts.DependencyDir, bin)
	}

	//nolint:gosec // G204: Subprocess launched with a potential tainted input or cmd arguments
	cmd := exec.CommandContext(ctx, bin, binArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = append(cmd.Env, os.Environ()...)

	for key, value := range opts.Envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	err = cmd.Run()
	out := ExecuteCommandOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: cmd.ProcessState.ExitCode(),
	}
	if err != nil {
		return out, runErr(stdout.String(), stderr.String(), err)
	}
	if out.ExitCode != 0 {
		return out, fmt.Errorf("got non-zero exit code, stdout [%q], stderr [%q]", stdout.String(), stderr.String())
	}
	return out, nil
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
