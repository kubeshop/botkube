package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const sampleCommunicationConfig = "testdata/comm_config.yaml"

func TestCommunicationConfigSuccess(t *testing.T) {
	t.Run("Load from file", func(t *testing.T) {
		// given
		t.Setenv("CONFIG_PATH", "testdata")

		var expConfig Communications
		loadYAMLFile(t, sampleCommunicationConfig, &expConfig)

		// when
		gotCfg, err := NewCommunicationsConfig()

		//then
		require.NoError(t, err)
		require.NotNil(t, gotCfg)
		assert.Equal(t, expConfig, *gotCfg)
	})

	t.Run("Load from file and override with environment variables", func(t *testing.T) {
		// given
		t.Setenv("CONFIG_PATH", "testdata")

		fixToken := fmt.Sprintf("TOKEN_FROM_ENV_%d", time.Now().Unix())
		t.Setenv("COMMUNICATIONS_SLACK_TOKEN", fixToken)
		var expConfig Communications
		loadYAMLFile(t, sampleCommunicationConfig, &expConfig)
		expConfig.Communications.Slack.Token = fixToken

		// when
		gotCfg, err := NewCommunicationsConfig()

		//then
		require.NoError(t, err)
		require.NotNil(t, gotCfg)
		assert.Equal(t, expConfig, *gotCfg)
	})
}

func loadYAMLFile(t *testing.T, path string, out interface{}) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)

	err = yaml.Unmarshal(raw, out)
	require.NoError(t, err)
}
