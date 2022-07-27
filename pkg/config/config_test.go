package config_test

import (
	"path/filepath"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/config"
)

// This test is based on golden file. To update golden file, run:
// go test -run=TestLoadConfigSuccess ./pkg/config/... -test.update-golden
func TestLoadConfigSuccess(t *testing.T) {
	// given
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SLACK_TOKEN", "token-from-env")
	t.Setenv("BOTKUBE_SETTINGS_CLUSTER__NAME", "cluster-name-from-env")
	t.Setenv("BOTKUBE_SETTINGS_KUBECONFIG", "kubeconfig-from-env")
	t.Setenv("BOTKUBE_SETTINGS_METRICS__PORT", "1313")

	// when
	gotCfg, _, err := config.LoadWithDefaults(func() []string {
		return []string{
			testdataFile(t, "config-all.yaml"),
			testdataFile(t, "config-global.yaml"),
			testdataFile(t, "config-slack-override.yaml"),
			testdataFile(t, "analytics.yaml"),
			testdataFile(t, "executors.yaml"),
		}
	})

	//then
	require.NoError(t, err)
	require.NotNil(t, gotCfg)
	gotData, err := yaml.Marshal(gotCfg)
	require.NoError(t, err)

	golden.Assert(t, string(gotData), filepath.Join(t.Name(), "config.golden.yaml"))
}

func TestFromEnvOrFlag(t *testing.T) {
	var expConfigPaths = []string{
		"configs/first.yaml",
		"configs/second.yaml",
		"configs/third.yaml",
	}

	t.Run("from envs variable only", func(t *testing.T) {
		// given
		t.Setenv("BOTKUBE_CONFIG_PATHS", "configs/first.yaml,configs/second.yaml,configs/third.yaml")

		// when
		gotConfigPaths := config.FromEnvOrFlag()

		// then
		assert.Equal(t, expConfigPaths, gotConfigPaths)
	})

	t.Run("from CLI flag only", func(t *testing.T) {
		// given
		fSet := pflag.NewFlagSet("testing", pflag.ContinueOnError)
		config.RegisterFlags(fSet)
		err := fSet.Parse([]string{"--config=configs/first.yaml,configs/second.yaml", "--config", "configs/third.yaml"})
		require.NoError(t, err)

		// when
		gotConfigPaths := config.FromEnvOrFlag()

		// then
		assert.Equal(t, expConfigPaths, gotConfigPaths)
	})

	t.Run("should honor env variable over the CLI flag", func(t *testing.T) {
		// given
		fSet := pflag.NewFlagSet("testing", pflag.ContinueOnError)
		config.RegisterFlags(fSet)

		err := fSet.Parse([]string{"--config=configs/from-cli-flag.yaml,configs/from-cli-flag-second.yaml"})
		require.NoError(t, err)

		t.Setenv("BOTKUBE_CONFIG_PATHS", "configs/first.yaml,configs/second.yaml,configs/third.yaml")

		// when
		gotConfigPaths := config.FromEnvOrFlag()

		// then
		assert.Equal(t, expConfigPaths, gotConfigPaths)
	})
}

func TestNormalizeConfigEnvName(t *testing.T) {
	// given
	tests := []struct {
		name            string
		givenEnvVarName string
		expYAMLKey      string
	}{
		{
			name:            "env var without any camel keys",
			givenEnvVarName: "BOTKUBE_SETTINGS_KUBECONFIG",
			expYAMLKey:      "settings.kubeconfig",
		},
		{
			name:            "env var with a camel key at the end",
			givenEnvVarName: "BOTKUBE_SETTINGS_METRICS__PORT",
			expYAMLKey:      "settings.metricsPort",
		},
		{
			name:            "env var with two camel keys at the end",
			givenEnvVarName: "BOTKUBE_COMMUNICATIONS_SLACK__TOKEN_ID__NAME",
			expYAMLKey:      "communications.slackToken.idName",
		},
		{
			name:            "env var with a camel key in the middle (2 words)",
			givenEnvVarName: "BOTKUBE_COMMUNICATIONS_SLACK__TOKEN_ID_NAME",
			expYAMLKey:      "communications.slackToken.id.name",
		},
		{
			name:            "env var with a camel key in the middle (3 words)",
			givenEnvVarName: "BOTKUBE_COMMUNICATIONS__SLACK__TOKEN_ID_NAME",
			expYAMLKey:      "communicationsSlackToken.id.name",
		},
		{
			name:            "multiple camel keys in the path",
			givenEnvVarName: "BOTKUBE_MY__COMMUNICATIONS_RANDOM__WORD_SLACK__TOKEN_ID_NAME",
			expYAMLKey:      "myCommunications.randomWord.slackToken.id.name",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotName := config.NormalizeConfigEnvName(tc.givenEnvVarName)

			// then
			assert.Equal(t, tc.expYAMLKey, gotName)
		})
	}
}

func TestLoadedConfigValidation(t *testing.T) {
	// given
	tests := []struct {
		name        string
		expErrMsg   string
		configFiles []string
	}{
		{
			name: "no config files",
			expErrMsg: heredoc.Doc(`
				while validating loaded configuration: 2 errors occurred:
					* Key: 'Config.Executors' Error:Field validation for 'Executors' failed on the 'required' tag
					* Key: 'Config.Communications' Error:Field validation for 'Communications' failed on the 'required' tag`),
			configFiles: nil,
		},
		{
			// TODO(remove): https://github.com/kubeshop/botkube/issues/596
			name: "empty executors and communications settings",
			expErrMsg: heredoc.Doc(`
				while validating loaded configuration: 2 errors occurred:
					* Key: 'Config.Executors' Error:Field validation for 'Executors' failed on the 'eq' tag
					* Key: 'Config.Communications' Error:Field validation for 'Communications' failed on the 'eq' tag`),
			configFiles: []string{
				testdataFile(t, "empty-executors-communications.yaml"),
			},
		},
		{
			// TODO(remove): https://github.com/kubeshop/botkube/issues/596
			name: "multiple executors and communications settings",
			expErrMsg: heredoc.Doc(`
				while validating loaded configuration: 2 errors occurred:
					* Key: 'Config.Executors' Error:Field validation for 'Executors' failed on the 'eq' tag
					* Key: 'Config.Communications' Error:Field validation for 'Communications' failed on the 'eq' tag`),
			configFiles: []string{
				testdataFile(t, "multiple-executors-communications.yaml"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, _, err := config.LoadWithDefaults(func() []string {
				return tc.configFiles
			})

			// then
			assert.Nil(t, cfg)
			assert.EqualError(t, err, tc.expErrMsg)
		})
	}
}

func testdataFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", t.Name(), name)
}
