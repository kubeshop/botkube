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

	intConfig "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/pkg/config"
)

// This test is based on golden file. To update golden file, run:
// go test -run=TestLoadConfigSuccess ./pkg/config/... -test.update-golden
func TestLoadConfigSuccess(t *testing.T) {
	// given
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SLACK_TOKEN", "xoxb-token-from-env")
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SOCKET__SLACK_BOT__TOKEN", "xoxb-token-from-env")
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SOCKET__SLACK_APP__TOKEN", "xapp-token-from-env")
	t.Setenv("BOTKUBE_SETTINGS_CLUSTER__NAME", "cluster-name-from-env")
	t.Setenv("BOTKUBE_SETTINGS_KUBECONFIG", "kubeconfig-from-env")
	t.Setenv("BOTKUBE_SETTINGS_METRICS__PORT", "1313")
	t.Setenv("BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL", "http://localhost:3000/botkube.yaml")

	// when
	gotCfg, _, err := config.LoadWithDefaults(func(*intConfig.GqlClient) ([]string, error) {
		return []string{
			testdataFile(t, "_aaa-special-file.yaml"),
			testdataFile(t, "config-all.yaml"),
			testdataFile(t, "config-global.yaml"),
			testdataFile(t, "config-slack-override.yaml"),
			testdataFile(t, "analytics.yaml"),
			testdataFile(t, "executors.yaml"),
			testdataFile(t, "actions.yaml"),
		}, nil
	}, nil)

	//then
	require.NoError(t, err)
	require.NotNil(t, gotCfg)
	gotData, err := yaml.Marshal(gotCfg)
	require.NoError(t, err)

	golden.Assert(t, string(gotData), filepath.Join(t.Name(), "config.golden.yaml"))
}

func TestLoadConfigWithPlugins(t *testing.T) {
	// given
	expSourcePlugin := config.PluginsExecutors{
		"botkube/keptn": {
			Enabled: true,
			Config: map[string]interface{}{
				"field": "value",
			},
		},
	}

	expExecutorPlugin := config.PluginsExecutors{
		"botkube/echo": {
			Enabled: true,
			Config: map[string]interface{}{
				"changeResponseToUpperCase": true,
			},
		},
	}

	// when
	gotCfg, _, err := config.LoadWithDefaults(func(*intConfig.GqlClient) ([]string, error) {
		return []string{
			testdataFile(t, "config-all.yaml"),
		}, nil
	}, nil)

	//then
	require.NoError(t, err)
	require.NotNil(t, gotCfg)

	assert.Equal(t, expSourcePlugin, gotCfg.Sources["k8s-events"].Plugins)
	assert.Equal(t, expExecutorPlugin, gotCfg.Executors["plugin-based"].Plugins)
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
		gotConfigPaths, err := config.FromProvider(nil)
		assert.NoError(t, err)

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
		gotConfigPaths, err := config.FromProvider(nil)
		assert.NoError(t, err)

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
		gotConfigPaths, err := config.FromProvider(nil)
		assert.NoError(t, err)

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

func TestLoadedConfigValidationErrors(t *testing.T) {
	// given
	tests := []struct {
		name        string
		expErrMsg   string
		configFiles []string
	}{
		{
			name: "no config files",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications' Communications is a required field`),
			configFiles: nil,
		},
		{
			name: "empty executors and communications settings",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications' Communications must contain at least 1 item`),
			configFiles: []string{
				testdataFile(t, "empty-executors-communications.yaml"),
			},
		},
		{
			name: "App token only",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken must have the xoxb- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#obtain-bot-token`),
			configFiles: []string{
				testdataFile(t, "app-token-only.yaml"),
			},
		},
		{
			name: "Bot token only",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken must have the xapp- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#generate-and-obtain-app-level-token`),
			configFiles: []string{
				testdataFile(t, "bot-token-only.yaml"),
			},
		},
		{
			name: "no tokens",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 5 errors occurred:
					* Key: 'Config.Communications[default-workspace].Slack.Token' Token is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken must have the xoxb- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#obtain-bot-token
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken must have the xapp- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#generate-and-obtain-app-level-token`),
			configFiles: []string{
				testdataFile(t, "no-token.yaml"),
			},
		},
		{
			name: "missing executor",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.kubectl-read-only' 'kubectl-read-only' binding not defined in Config.Executors`),
			configFiles: []string{
				testdataFile(t, "missing-executor.yaml"),
			},
		},
		{
			name: "missing source",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.k8s-events' 'k8s-events' binding not defined in Config.Sources`),
			configFiles: []string{
				testdataFile(t, "missing-source.yaml"),
			},
		},
		{
			name: "missing action bindings",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Actions[show-created-resource].Bindings.k8s-err-events' 'k8s-err-events' binding not defined in Config.Sources
					* Key: 'Config.Actions[show-created-resource].Bindings.kubectl-read-only' 'kubectl-read-only' binding not defined in Config.Executors`),
			configFiles: []string{
				testdataFile(t, "missing-action-bindings.yaml"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, details, err := config.LoadWithDefaults(func(*intConfig.GqlClient) ([]string, error) {
				return tc.configFiles, nil
			}, nil)

			// then
			assert.Nil(t, cfg)
			assert.NoError(t, details.ValidateWarnings)
			assert.EqualError(t, err, tc.expErrMsg)
		})
	}
}

func TestLoadedConfigValidationWarnings(t *testing.T) {
	// given
	tests := []struct {
		name        string
		expWarnMsg  string
		configFiles []string
	}{
		{
			name: "executor specifies all and exact namespace in include property",
			expWarnMsg: heredoc.Doc(`
				2 errors occurred:
					* Key: 'Config.Sources[k8s-events].Kubernetes.Resources[0].Namespaces.Include' Include matches both all and exact namespaces
					* Key: 'Config.Executors[kubectl-read-only].Kubectl.Namespaces.Include' Include matches both all and exact namespaces`),
			configFiles: []string{
				testdataFile(t, "executors-include-warning.yaml"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, details, err := config.LoadWithDefaults(func(*intConfig.GqlClient) ([]string, error) {
				return tc.configFiles, nil
			}, nil)

			// then
			assert.NotNil(t, cfg)
			assert.NoError(t, err)
			assert.EqualError(t, details.ValidateWarnings, tc.expWarnMsg)
		})
	}
}

func TestLoadedConfigEnabledPluginErrors(t *testing.T) {
	// given
	tests := []struct {
		name        string
		expErrMsg   string
		configFiles []string
	}{
		{
			name: "should report an issue with bindings for same plugin name but coming from two different repositories",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.mszostok/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin from "botkube" repository. Bind it to a different channel, change it to the one from the "botkube" repository, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.mszostok/kubectl@v1.0.0' conflicts with already bound "kubectl" plugin from "botkube" repository. Bind it to a different channel, change it to the one from the "botkube" repository, or remove it.`),
			configFiles: []string{
				testdataFile(t, "bind-diff-repo.yaml"),
			},
		},
		{
			name: "should report an issue with bindings for plugins coming from the same repository but one refers to the latest version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[latest].Bindings.botkube/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin in the latest version. Bind it to a different channel, change it to the latest version, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[latest].Bindings.botkube/kubectl@v1.0.0' conflicts with already bound "kubectl" plugin in the latest version. Bind it to a different channel, change it to the latest version, or remove it.`),
			configFiles: []string{
				testdataFile(t, "bind-diff-ver-latest.yaml"),
			},
		},
		{
			name: "should report an issue with bindings for plugins coming from the same repository but with different version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[versions].Bindings.botkube/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin in the "v1.0.0" version. Bind it to a different channel, change it to the "v1.0.0" version, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[versions].Bindings.botkube/kubectl@v2.0.0' conflicts with already bound "kubectl" plugin in the "v1.0.0" version. Bind it to a different channel, change it to the "v1.0.0" version, or remove it.`),
			configFiles: []string{
				testdataFile(t, "bind-diff-ver.yaml"),
			},
		},
		{
			name: "should report an issue with source configuration group that imports the same plugin from different repositories",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Sources[duplicated-names].mszostok/prometheus@v1.2.0' conflicts with already defined "prometheus" plugin from "botkube" repository. Extract it to a dedicated configuration group or remove it from this one.
					* Key: 'Config.Executors[duplicated-names].mszostok/kubectl' conflicts with already defined "kubectl" plugin from "botkube" repository. Extract it to a dedicated configuration group or remove it from this one.`),
			configFiles: []string{
				testdataFile(t, "cfg-group-diff-repo.yaml"),
			},
		},
		{
			name: "should report an issue with source configuration group that imports plugins from the same repository but with different version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Sources[wrong-vers].botkube/prometheus@v1.2.0' conflicts with already defined "prometheus" plugin in the "v1.0.0" version. Extract it to a dedicated configuration group or remove it from this one.
					* Key: 'Config.Executors[wrong-vers].botkube/kubectl@v1.0.0' conflicts with already defined "kubectl" plugin in the latest version. Extract it to a dedicated configuration group or remove it from this one.`),
			configFiles: []string{
				testdataFile(t, "cfg-group-diff-ver.yaml"),
			},
		},
		{
			name: "should report an issue with source configuration group that imports plugins with wrong syntax",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 7 errors occurred:
					* Key: 'Config.Sources[wrong-name]./@v1.2.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 2 errors occurred:
						* repository name is required
						* plugin name is required
					* Key: 'Config.Sources[wrong-name]./prometheus@v1.2.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* repository name is required
					* Key: 'Config.Sources[wrong-name].botkube/@v1.0.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* plugin name is required
					* Key: 'Config.Executors[wrong-name]./@v1.0.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 2 errors occurred:
						* repository name is required
						* plugin name is required
					* Key: 'Config.Executors[wrong-name]./kubectl@v1.0.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* repository name is required
					* Key: 'Config.Executors[wrong-name].some-3rd-plugin' plugin key "some-3rd-plugin" doesn't follow the required {repo_name}/{plugin_name} syntax
					* Key: 'Config.Executors[wrong-name].testing/@v1.0.0' doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* plugin name is required`),
			configFiles: []string{
				testdataFile(t, "cfg-group-wrong-plugin-def.yaml"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, details, err := config.LoadWithDefaults(func(*intConfig.GqlClient) ([]string, error) {
				return tc.configFiles, nil
			}, nil)

			// then
			assert.Nil(t, cfg)
			assert.NoError(t, details.ValidateWarnings)
			assert.Equal(t, tc.expErrMsg, err.Error())
		})
	}
}

func testdataFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", t.Name(), name)
}

func TestIsNamespaceAllowed(t *testing.T) {
	tests := map[string]struct {
		nsConfig  config.Namespaces
		givenNs   string
		isAllowed bool
	}{
		"should watch all except ignored ones": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: []string{"demo", "abc"}},
			givenNs:   "demo",
			isAllowed: false,
		},
		"should watch all when ignore has empty items only": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: []string{""}},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should watch all when ignore is a nil slice": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: nil},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should ignore matched by regex": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: []string{"my-.*"}},
			givenNs:   "my-ns",
			isAllowed: false,
		},
		"should ignore matched by regexp even if exact name is mentioned too": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: []string{"demo", "ignored-.*-ns"}},
			givenNs:   "ignored-42-ns",
			isAllowed: false,
		},
		"should watch all if regexp is not matching given namespace": {
			nsConfig:  config.Namespaces{Include: []string{".*"}, Exclude: []string{"demo-.*"}},
			givenNs:   "demo",
			isAllowed: true,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual := test.nsConfig.IsAllowed(test.givenNs)
			if actual != test.isAllowed {
				t.Errorf("expected: %v != actual: %v\n", test.isAllowed, actual)
			}
		})
	}
}

func TestSortCfgFiles(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
	}{
		"No special files": {
			input:    []string{"config.yaml", ".bar.yaml", "/_foo/bar.yaml", "/_bar/baz.yaml"},
			expected: []string{"config.yaml", ".bar.yaml", "/_foo/bar.yaml", "/_bar/baz.yaml"},
		},
		"Special files": {
			input:    []string{"_test.yaml", "config.yaml", "_foo.yaml", ".bar.yaml", "/bar/_baz.yaml"},
			expected: []string{"config.yaml", ".bar.yaml", "_test.yaml", "_foo.yaml", "/bar/_baz.yaml"},
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual := config.SortCfgFiles(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
