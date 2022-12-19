package config

import "context"

// Provider provider for configuration sources
type Provider interface {
	Configs(ctx context.Context) ([]string, error)
}
