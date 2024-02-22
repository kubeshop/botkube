package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MakeNowJust/heredoc"
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
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SLACK_TOKEN", "xoxb-token-from-env")
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SOCKET__SLACK_BOT__TOKEN", "xoxb-token-from-env")
	t.Setenv("BOTKUBE_COMMUNICATIONS_DEFAULT-WORKSPACE_SOCKET__SLACK_APP__TOKEN", "xapp-token-from-env")
	t.Setenv("BOTKUBE_SETTINGS_CLUSTER__NAME", "cluster-name-from-env")
	t.Setenv("BOTKUBE_SETTINGS_KUBECONFIG", "kubeconfig-from-env")
	t.Setenv("BOTKUBE_SETTINGS_METRICS__PORT", "1313")
	t.Setenv("BOTKUBE_SETTINGS_HEALTH__PORT", "1314")
	t.Setenv("BOTKUBE_PLUGINS_REPOSITORIES_BOTKUBE_URL", "http://localhost:3000/botkube.yaml")

	// when
	files := config.YAMLFiles{
		readTestdataFile(t, "config-all.yaml"),
		readTestdataFile(t, "config-global.yaml"),
		readTestdataFile(t, "config-slack-override.yaml"),
		readTestdataFile(t, "analytics.yaml"),
		readTestdataFile(t, "executors.yaml"),
		readTestdataFile(t, "actions.yaml"),
		readTestdataFile(t, "_aaa-special-file.yaml"),
	}
	// sorted was extracted ...
	gotCfg, _, err := config.LoadWithDefaults(files)

	//then
	require.NoError(t, err)
	require.NotNil(t, gotCfg)
	gotData, err := yaml.Marshal(gotCfg)
	require.NoError(t, err)

	golden.Assert(t, string(gotData), filepath.Join(t.Name(), "config.golden.yaml"))
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
		name      string
		expErrMsg string
		configs   [][]byte
		isWarning bool
	}{
		{
			name: "no config files",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications' Communications is a required field`),
			configs: nil,
		},
		{
			name: "empty executors and communications settings",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications' Communications must contain at least 1 item`),
			configs: [][]byte{
				readTestdataFile(t, "empty-executors-communications.yaml"),
			},
		},
		{
			name: "App token only",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken must have the xoxb- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#obtain-bot-token`),
			configs: [][]byte{
				readTestdataFile(t, "app-token-only.yaml"),
			},
		},
		{
			name: "Bot token only",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken must have the xapp- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#generate-and-obtain-app-level-token`),
			configs: [][]byte{
				readTestdataFile(t, "bot-token-only.yaml"),
			},
		},
		{
			name: "no tokens",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 4 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken is a required field
					* Key: 'Config.Communications[default-workspace].SocketSlack.BotToken' BotToken must have the xoxb- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#obtain-bot-token
					* Key: 'Config.Communications[default-workspace].SocketSlack.AppToken' AppToken must have the xapp- prefix. Learn more at https://docs.botkube.io/installation/socketslack/#generate-and-obtain-app-level-token`),
			configs: [][]byte{
				readTestdataFile(t, "no-token.yaml"),
			},
		},
		{
			name: "missing executor",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.kubectl-read-only' 'kubectl-read-only' binding not defined in Config.Executors`),
			configs: [][]byte{
				readTestdataFile(t, "missing-executor.yaml"),
			},
		},
		{
			name: "missing source",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.k8s-events' 'k8s-events' binding not defined in Config.Sources`),
			configs: [][]byte{
				readTestdataFile(t, "missing-source.yaml"),
			},
		},
		{
			name: "missing action bindings",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Actions[show-created-resource].Bindings.k8s-err-events' 'k8s-err-events' binding not defined in Config.Sources
					* Key: 'Config.Actions[show-created-resource].Bindings.kubectl-read-only' 'kubectl-read-only' binding not defined in Config.Executors`),
			configs: [][]byte{
				readTestdataFile(t, "missing-action-bindings.yaml"),
			},
		},
		{
			name: "missing alias command",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Aliases[eee].Command' Command is a required field`),
			configs: [][]byte{
				readTestdataFile(t, "missing-alias-command.yaml"),
			},
		},
		{
			name: "invalid alias command",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Aliases[foo].Command' Command prefix 'foo' not found in executors or builtin commands`),
			configs: [][]byte{
				readTestdataFile(t, "invalid-alias-command.yaml"),
			},
		},
		{
			name: "RBAC helm executors are different",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-group].SocketSlack.Channels[botkube].Bindings.helm-1' Binding is referencing plugins of same kind with different RBAC. 'helm-1' and 'helm-2' bindings must be identical when used together.`),
			configs: [][]byte{
				readTestdataFile(t, "executors-rbac.yaml"),
			},
		},
		{
			name: "RBAC cm sources are different",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 1 error occurred:
					* Key: 'Config.Communications[default-group].SocketSlack.Channels[botkube].Bindings.cm-1' Binding is referencing plugins of same kind with different RBAC. 'cm-1' and 'cm-2' bindings must be identical when used together.`),
			configs: [][]byte{
				readTestdataFile(t, "sources-rbac.yaml"),
			},
		},
		{
			name: "Invalid channel names",
			expErrMsg: heredoc.Doc(`
				4 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.alias2.Name' The channel name 'INCORRECT' seems to be invalid. See the documentation to learn more: https://api.slack.com/methods/conversations.rename#naming.
					* Key: 'Config.Communications[default-workspace].CloudSlack.alias2.Name' The channel name 'INCORRECT' seems to be invalid. See the documentation to learn more: https://api.slack.com/methods/conversations.rename#naming.
					* Key: 'Config.Communications[default-workspace].Mattermost.alias.Name' The channel name 'too-long name really really really really really really really really really really really really long' seems to be invalid. See the documentation to learn more: https://docs.mattermost.com/channels/channel-naming-conventions.html.
					* Key: 'Config.Communications[default-workspace].Discord.alias2.ID' The channel name 'incorrect' seems to be invalid. See the documentation to learn more: https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-.`),
			configs: [][]byte{
				readTestdataFile(t, "invalid-channels.yaml"),
			},
			isWarning: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, details, err := config.LoadWithDefaults(tc.configs)

			// then
			if tc.isWarning {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Error(t, details.ValidateWarnings)
				assert.EqualError(t, details.ValidateWarnings, tc.expErrMsg)
				return
			}

			assert.Nil(t, cfg)
			assert.NoError(t, details.ValidateWarnings)
			assert.EqualError(t, err, tc.expErrMsg)
		})
	}
}

func TestLoadedConfigEnabledPluginErrors(t *testing.T) {
	// given
	tests := []struct {
		name      string
		expErrMsg string
		configs   [][]byte
	}{
		{
			name: "should report an issue with bindings for same plugin name but coming from two different repositories",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.mszostok/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin from "botkube" repository. Bind it to a different channel, change it to the one from the "botkube" repository, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[alias].Bindings.mszostok/kubectl@v1.0.0' conflicts with already bound "kubectl" plugin from "botkube" repository. Bind it to a different channel, change it to the one from the "botkube" repository, or remove it.`),
			configs: [][]byte{
				readTestdataFile(t, "bind-diff-repo.yaml"),
			},
		},
		{
			name: "should report an issue with bindings for plugins coming from the same repository but one refers to the latest version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[latest].Bindings.botkube/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin in the latest version. Bind it to a different channel, change it to the latest version, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[latest].Bindings.botkube/kubectl@v1.0.0' conflicts with already bound "kubectl" plugin in the latest version. Bind it to a different channel, change it to the latest version, or remove it.`),
			configs: [][]byte{
				readTestdataFile(t, "bind-diff-ver-latest.yaml"),
			},
		},
		{
			name: "should report an issue with bindings for plugins coming from the same repository but with different version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[versions].Bindings.botkube/prometheus@v1.2.0' conflicts with already bound "prometheus" plugin in the "v1.0.0" version. Bind it to a different channel, change it to the "v1.0.0" version, or remove it.
					* Key: 'Config.Communications[default-workspace].SocketSlack.Channels[versions].Bindings.botkube/kubectl@v2.0.0' conflicts with already bound "kubectl" plugin in the "v1.0.0" version. Bind it to a different channel, change it to the "v1.0.0" version, or remove it.`),
			configs: [][]byte{
				readTestdataFile(t, "bind-diff-ver.yaml"),
			},
		},
		{
			name: "should report an issue with source configuration group that imports the same plugin from different repositories",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Sources[duplicated-names].mszostok/prometheus@v1.2.0' conflicts with already defined "prometheus" plugin from "botkube" repository. Extract it to a dedicated configuration group or remove it from this one.
					* Key: 'Config.Executors[duplicated-names].mszostok/kubectl' conflicts with already defined "kubectl" plugin from "botkube" repository. Extract it to a dedicated configuration group or remove it from this one.`),
			configs: [][]byte{
				readTestdataFile(t, "cfg-group-diff-repo.yaml"),
			},
		},
		{
			name: "should report an issue with source configuration group that imports plugins from the same repository but with different version",
			expErrMsg: heredoc.Doc(`
				found critical validation errors: 2 errors occurred:
					* Key: 'Config.Sources[wrong-vers].botkube/prometheus@v1.2.0' conflicts with already defined "prometheus" plugin in the "v1.0.0" version. Extract it to a dedicated configuration group or remove it from this one.
					* Key: 'Config.Executors[wrong-vers].botkube/kubectl@v1.0.0' conflicts with already defined "kubectl" plugin in the latest version. Extract it to a dedicated configuration group or remove it from this one.`),
			configs: [][]byte{
				readTestdataFile(t, "cfg-group-diff-ver.yaml"),
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
			configs: [][]byte{
				readTestdataFile(t, "cfg-group-wrong-plugin-def.yaml"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg, details, err := config.LoadWithDefaults(tc.configs)

			// then
			assert.Nil(t, cfg)
			assert.NoError(t, details.ValidateWarnings)
			assert.Equal(t, tc.expErrMsg, err.Error())
		})
	}
}

func readTestdataFile(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", t.Name(), name)
	out, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	return out
}

func TestRegexConstraints_IsAllowed(t *testing.T) {
	tests := map[string]struct {
		nsConfig           config.RegexConstraints
		givenNs            string
		isAllowed          bool
		expectedErrMessage string
	}{
		"should match all except ignored ones": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"demo", "abc"}},
			givenNs:   "demo",
			isAllowed: false,
		},
		"should match all when ignore has empty items only": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{""}},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should match all when ignore is a nil slice": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: nil},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should ignore matched by regex": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"my-.*"}},
			givenNs:   "my-ns",
			isAllowed: false,
		},
		"should ignore matched by regexp even if exact name is mentioned too": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"demo", "ignored-.*-ns"}},
			givenNs:   "ignored-42-ns",
			isAllowed: false,
		},
		"should match all if regexp is not matching given namespace": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"demo-.*"}},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should match empty value": {
			nsConfig:  config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"demo-.*"}},
			givenNs:   "",
			isAllowed: true,
		},
		"should match only empty value": {
			nsConfig:  config.RegexConstraints{Include: []string{"^$"}, Exclude: []string{}},
			givenNs:   "",
			isAllowed: true,
		},
		"invalid exclude regex": {
			nsConfig:           config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"["}},
			givenNs:            "demo",
			isAllowed:          false,
			expectedErrMessage: "while matching \"demo\" with exclude regex \"[\": error parsing regexp: missing closing ]: `[`",
		},
		"invalid include regex": {
			nsConfig:           config.RegexConstraints{Include: []string{"["}, Exclude: []string{}},
			givenNs:            "demo",
			isAllowed:          false,
			expectedErrMessage: "while matching \"demo\" with include regex \"[\": error parsing regexp: missing closing ]: `[`",
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual, err := test.nsConfig.IsAllowed(test.givenNs)

			if test.expectedErrMessage != "" {
				require.False(t, actual)
				require.EqualError(t, err, test.expectedErrMessage)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.isAllowed, actual)
		})
	}
}
