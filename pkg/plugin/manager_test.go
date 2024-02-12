package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestCollectEnabledRepositories(t *testing.T) {
	tests := []struct {
		name string

		enabledExecutors    []string
		enabledSources      []string
		definedRepositories map[string]config.PluginsRepository

		expErrMsg string
	}{
		{
			name: "report not defined repositories for source plugins",
			enabledSources: []string{
				"botkube/cm-watcher",
				"mszostok/hakuna-matata",
			},
			expErrMsg: heredoc.Doc(`
				2 errors occurred:
					* repository "botkube" is not defined, but it is referred by source plugin called "botkube/cm-watcher"
					* repository "mszostok" is not defined, but it is referred by source plugin called "mszostok/hakuna-matata"`),
		},
		{
			name: "report not defined repositories for executor plugins",
			enabledExecutors: []string{
				"botkube/helm",
				"botkube/kubectl",
				"mszostok/hakuna-matata",
			},
			expErrMsg: heredoc.Doc(`
				3 errors occurred:
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/helm"
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/kubectl"
					* repository "mszostok" is not defined, but it is referred by executor plugin called "mszostok/hakuna-matata"`),
		},
		{
			name: "report not defined repositories for source and executor plugins",
			enabledSources: []string{
				"botkube/cm-watcher",
				"mszostok/hakuna-matata",
			},
			enabledExecutors: []string{
				"botkube/helm",
				"botkube/kubectl",
				"mszostok/hakuna-matata",
			},
			expErrMsg: heredoc.Doc(`
				5 errors occurred:
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/helm"
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/kubectl"
					* repository "mszostok" is not defined, but it is referred by executor plugin called "mszostok/hakuna-matata"
					* repository "botkube" is not defined, but it is referred by source plugin called "botkube/cm-watcher"
					* repository "mszostok" is not defined, but it is referred by source plugin called "mszostok/hakuna-matata"`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			manager := NewManager(loggerx.NewNoop(), config.Logger{}, config.PluginManagement{
				Repositories: tc.definedRepositories,
			}, tc.enabledExecutors, tc.enabledSources, make(chan string), NewHealthStats(1))

			// when
			out, err := manager.collectEnabledRepositories()

			// then
			assert.Empty(t, out)
			assert.EqualError(t, err, tc.expErrMsg)
		})
	}
}

func TestNewPluginOSRunCommand_HappyPath(t *testing.T) {
	// given
	path := "/tmp/plugins/executor_v0.1.0_helm"
	depsPath := "/tmp/plugins/executor_v0.1.0_helm_deps"
	expectedEnvValue := fmt.Sprintf("PLUGIN_DEPENDENCY_DIR=%s", depsPath)

	// when
	actual := newPluginOSRunCommand(path)

	// then
	assert.Equal(t, path, actual.Path)
	var found bool
	for _, env := range actual.Env {
		if !strings.HasPrefix(env, "PLUGIN_DEPENDENCY_DIR=") {
			continue
		}
		assert.Equal(t, expectedEnvValue, env)
		found = true
		break
	}
	assert.True(t, found)
}

func TestManager_RenderPluginIndexHeaders(t *testing.T) {
	// given
	remoteCfg := remote.Config{
		Endpoint:   "http://endpoint",
		Identifier: "identifier",
		APIKey:     "api-key",
	}

	for _, testCase := range []struct {
		Name           string
		InHeaders      map[string]string
		ExpectedOut    map[string]string
		ExpectedErrMsg string
	}{
		{
			Name: "Success",
			InHeaders: map[string]string{
				"API-Key":    "{{ .Remote.APIKey }}",
				"URL":        "{{ .Remote.Endpoint }}",
				"Identifier": "{{ .Remote.Identifier }}",
				"Combined":   "{{ .Remote.Identifier }} / {{ .Remote.APIKey }}",
				"Static":     "Value",
			},
			ExpectedOut: map[string]string{
				"API-Key":    remoteCfg.APIKey,
				"URL":        remoteCfg.Endpoint,
				"Identifier": remoteCfg.Identifier,
				"Combined":   remoteCfg.Identifier + " / " + remoteCfg.APIKey,
				"Static":     "Value",
			},
		},
		{
			Name: "Error",
			InHeaders: map[string]string{
				"Err": "{{ .Remote.ID }}",
			},
			ExpectedErrMsg: heredoc.Doc(`
				1 error occurred:
					* while rendering header "Err": while rendering string "{{ .Remote.ID }}": template: tpl:1:10: executing "tpl" at <.Remote.ID>: can't evaluate field ID in type remote.Config`),
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			manager := &Manager{
				indexRenderData: IndexRenderData{
					Remote: remoteCfg,
				},
			}

			// when
			out, err := manager.renderPluginIndexHeaders(testCase.InHeaders)

			// then
			if testCase.ExpectedErrMsg != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.ExpectedOut, out)
		})
	}
}
