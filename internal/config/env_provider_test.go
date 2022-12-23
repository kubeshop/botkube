package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvProviderSuccess(t *testing.T) {
	//given
	t.Setenv("BOTKUBE_CONFIG_PATHS", "testdata/TestEnvProviderSuccess/config.yaml")

	// when
	p := NewEnvProvider()
	configs, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	content, err := os.ReadFile("testdata/TestEnvProviderSuccess/config.yaml")
	assert.NoError(t, err)
	assert.Equal(t, content, configs[0])
}

func TestEnvProviderErr(t *testing.T) {
	// when
	p := NewEnvProvider()
	_, err := p.Configs(context.Background())

	// then
	assert.Equal(t, "while reading a file: read .: is a directory", err.Error())
}
