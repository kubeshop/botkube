package config

import (
	"context"
)

type StaticProvider struct {
	Files []string
}

func NewStaticProvider(configs []string) *StaticProvider {
	return &StaticProvider{Files: configs}
}
func (e *StaticProvider) Configs(ctx context.Context) ([]string, error) {
	return e.Files, nil
}
