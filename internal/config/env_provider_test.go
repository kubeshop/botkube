package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvProviderSuccess(t *testing.T) {
	//given
	t.Setenv("BOTKUBE_CONFIG_PATHS", "testdata/TestEnvProviderSuccess/config.yaml")

	// when
	p := NewEnvProvider()
	configs, cfgVer, err := p.Configs(context.Background())

	// then
	require.NoError(t, err)
	content, err := os.ReadFile("testdata/TestEnvProviderSuccess/config.yaml")
	assert.NoError(t, err)
	assert.Equal(t, content, configs[0])
	assert.Equal(t, cfgVer, 0)
}

func TestEnvProviderErr(t *testing.T) {
	// when
	p := NewEnvProvider()
	_, _, err := p.Configs(context.Background())

	// then
	assert.Equal(t, "while reading a file: read .: is a directory", err.Error())
}
