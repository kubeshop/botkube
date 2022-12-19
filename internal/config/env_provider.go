package config

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// EnvProvider environment config source provider
type EnvProvider struct {
}

// NewEnvProvider initializes new environment config source provider
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Configs returns list of config file locations
func (e *EnvProvider) Configs(ctx context.Context) ([]string, error) {
	envCfgs := os.Getenv("BOTKUBE_CONFIG_PATHS")
	if envCfgs != "" {
		return strings.Split(envCfgs, ","), nil
	}
	return nil, errors.New("failed to get config files from environment variable")
}
