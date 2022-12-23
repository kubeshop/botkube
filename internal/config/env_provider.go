package config

import (
	"context"
	"os"
	"strings"
)

// EnvProvider environment config source provider
type EnvProvider struct {
}

// NewEnvProvider initializes new environment config source provider
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Configs returns list of config file locations
func (e *EnvProvider) Configs(ctx context.Context) (YAMLFiles, error) {
	envCfgs := os.Getenv("BOTKUBE_CONFIG_PATHS")
	configPaths := strings.Split(envCfgs, ",")

	return NewFileSystemProvider(configPaths).Configs(ctx)
}
