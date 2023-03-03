package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/config"
)

func TestGetProvider(t *testing.T) {
	t.Run("from envs variable only", func(t *testing.T) {
		// given
		t.Setenv("BOTKUBE_CONFIG_PATHS", "testdata/TestGetProvider/first.yaml,testdata/TestGetProvider/second.yaml,testdata/TestGetProvider/third.yaml")

		// when
		provider := config.GetProvider(false, nil)
		gotConfigs, _, err := provider.Configs(context.Background())
		assert.NoError(t, err)

		// then
		c, err := os.ReadFile("testdata/TestGetProvider/all.yaml")
		assert.NoError(t, err)
		assert.Equal(t, c, gotConfigs.Merge())
	})

	t.Run("from CLI flag only", func(t *testing.T) {
		// given
		fSet := pflag.NewFlagSet("testing", pflag.ContinueOnError)
		config.RegisterFlags(fSet)
		err := fSet.Parse([]string{"--config=testdata/TestGetProvider/first.yaml,testdata/TestGetProvider/second.yaml", "--config", "testdata/TestGetProvider/third.yaml"})
		require.NoError(t, err)

		// when
		provider := config.GetProvider(false, nil)
		gotConfigs, _, err := provider.Configs(context.Background())
		assert.NoError(t, err)

		// then
		c, err := os.ReadFile("testdata/TestGetProvider/all.yaml")
		assert.NoError(t, err)
		assert.Equal(t, c, gotConfigs.Merge())
	})

	t.Run("should honor env variable over the CLI flag", func(t *testing.T) {
		// given
		fSet := pflag.NewFlagSet("testing", pflag.ContinueOnError)
		config.RegisterFlags(fSet)

		err := fSet.Parse([]string{"--config=testdata/TestGetProvider/from-cli-flag.yaml,testdata/TestGetProvider/from-cli-flag-second.yaml"})
		require.NoError(t, err)

		t.Setenv("BOTKUBE_CONFIG_PATHS", "testdata/TestGetProvider/first.yaml,testdata/TestGetProvider/second.yaml,testdata/TestGetProvider/third.yaml")

		// when
		provider := config.GetProvider(false, nil)
		gotConfigs, _, err := provider.Configs(context.Background())
		assert.NoError(t, err)

		// then
		c, err := os.ReadFile("testdata/TestGetProvider/all.yaml")
		assert.NoError(t, err)
		assert.Equal(t, c, gotConfigs.Merge())
	})
}
