package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestCollectEnabledRepositories(t *testing.T) {
	tests := []struct {
		name string

		enabledExecutors    []Plugin
		enabledSources      []Plugin
		definedRepositories map[string]config.PluginsRepositories

		expErrMsg string
	}{
		{
			name: "report not defined repositories for source plugins",
			enabledSources: []Plugin{
				{Name: "botkube/cm-watcher"},
				{Name: "mszostok/hakuna-matata"},
			},
			expErrMsg: heredoc.Doc(`
				2 errors occurred:
					* repository "botkube" is not defined, but it is referred by source plugin called "botkube/cm-watcher"
					* repository "mszostok" is not defined, but it is referred by source plugin called "mszostok/hakuna-matata"`),
		},
		{
			name: "report not defined repositories for executor plugins",
			enabledExecutors: []Plugin{
				{Name: "botkube/helm"},
				{Name: "botkube/kubectl"},
				{Name: "mszostok/hakuna-matata"},
			},
			expErrMsg: heredoc.Doc(`
				3 errors occurred:
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/helm"
					* repository "botkube" is not defined, but it is referred by executor plugin called "botkube/kubectl"
					* repository "mszostok" is not defined, but it is referred by executor plugin called "mszostok/hakuna-matata"`),
		},
		{
			name: "report not defined repositories for source and executor plugins",
			enabledSources: []Plugin{
				{Name: "botkube/cm-watcher"},
				{Name: "mszostok/hakuna-matata"},
			},
			enabledExecutors: []Plugin{
				{Name: "botkube/helm"},
				{Name: "botkube/kubectl"},
				{Name: "mszostok/hakuna-matata"},
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
			}, tc.enabledExecutors, tc.enabledSources, make(chan string))

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
