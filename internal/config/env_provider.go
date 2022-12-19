package config

import (
	"context"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type EnvProvider struct {
}

func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}
func (e *EnvProvider) Configs(ctx context.Context) ([]string, error) {
	envCfgs := os.Getenv("BOTKUBE_CONFIG_PATHS")
	if envCfgs != "" {
		return strings.Split(envCfgs, ","), nil
	}
	return nil, errors.New("failed to get config files from environment variable.")
}
