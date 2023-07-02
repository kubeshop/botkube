package pluginx

// ExecuteCommandOptions represents the options for executing a command.
type ExecuteCommandOptions struct {
	Envs          map[string]string
	DependencyDir string
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
