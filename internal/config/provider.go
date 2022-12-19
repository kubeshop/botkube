package config

import "context"

type Provider interface {
	Configs(ctx context.Context) ([]string, error)
}
