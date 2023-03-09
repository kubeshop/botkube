package config

import (
	"bytes"
	"context"
)

// YAMLFiles denotes list of configurations in bytes
type YAMLFiles [][]byte

// Merge flattens 2d config bytes
func (y YAMLFiles) Merge() []byte {
	return bytes.Join(y, nil)
}

// Provider for configuration sources
type Provider interface {
	Configs(ctx context.Context) (YAMLFiles, int, error)
}
