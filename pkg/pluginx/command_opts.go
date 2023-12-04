package pluginx

import (
	"io"
)

// ExecuteCommandOptions represents the options for executing a command.
type ExecuteCommandOptions struct {
	Envs            map[string]string
	DependencyDir   string
	WorkDir         string
	Stdin           io.Reader
	ClearColorCodes bool
}

// ExecuteCommandMutation is a function type that can be used to modify ExecuteCommandOptions.
type ExecuteCommandMutation func(*ExecuteCommandOptions)

// ExecuteCommandEnvs is a function that sets the environment variables.
func ExecuteCommandEnvs(envs map[string]string) ExecuteCommandMutation {
	return func(options *ExecuteCommandOptions) {
		options.Envs = envs
	}
}

// ExecuteCommandDependencyDir is a function that sets the dependency directory.
func ExecuteCommandDependencyDir(dir string) ExecuteCommandMutation {
	return func(options *ExecuteCommandOptions) {
		options.DependencyDir = dir
	}
}

// ExecuteCommandWorkingDir is a function that sets the working directory of the command.
func ExecuteCommandWorkingDir(dir string) ExecuteCommandMutation {
	return func(options *ExecuteCommandOptions) {
		options.WorkDir = dir
	}
}

// ExecuteCommandStdin is a function that sets the stdin of the command.
func ExecuteCommandStdin(in io.Reader) ExecuteCommandMutation {
	return func(options *ExecuteCommandOptions) {
		options.Stdin = in
	}
}

// ExecuteClearColorCodes is a function that enables removing color codes.
func ExecuteClearColorCodes() ExecuteCommandMutation {
	return func(options *ExecuteCommandOptions) {
		options.ClearColorCodes = true
	}
}
