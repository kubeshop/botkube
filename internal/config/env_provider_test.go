package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvProviderSuccess(t *testing.T) {
	//given
	t.Setenv("BOTKUBE_CONFIG_PATHS", "/tmp/a.yaml")

	// when
	p := NewEnvProvider()
	configs, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	assert.Equal(t, []string{"/tmp/a.yaml"}, configs)
}

func TestEnvProviderErr(t *testing.T) {
	// when
	p := NewEnvProvider()
	_, err := p.Configs(context.Background())

	// then
	assert.Equal(t, "failed to get config files from environment variable", err.Error())
}
