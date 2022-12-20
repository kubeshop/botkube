package config

import "context"

// Provider for configuration sources
type Provider interface {
	Configs(ctx context.Context) ([]string, error)
}
