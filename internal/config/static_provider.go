package config

import (
	"context"
)

// StaticProvider allows consumer to pass config files statically
type StaticProvider struct {
	Files []string
}

// NewStaticProvider initializes new static config source provider
func NewStaticProvider(configs []string) *StaticProvider {
	return &StaticProvider{Files: configs}
}

// Configs returns list of config file locations.
func (e *StaticProvider) Configs(ctx context.Context) ([]string, error) {
	return e.Files, nil
}
