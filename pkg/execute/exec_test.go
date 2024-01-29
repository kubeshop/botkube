package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestExecutorBindingsExecutor(t *testing.T) {
	testCases := []struct {
		name     string
		cfg      config.Config
		bindings []string

		expOutput string
	}{
		{
			name: "two execs",
			cfg: config.Config{
				Executors: map[string]config.Executors{
					"kubectl-team-a": {
						Plugins: map[string]config.Plugin{
							"botkube/kubectl": {
								Enabled: true,
							},
						},
					},
					"kubectl-team-b": {
						Plugins: map[string]config.Plugin{
							"botkube/echo": {
								Enabled: true,
							},
						},
					},
				},
				Aliases: map[string]config.Alias{
					"k": {
						Command: "kubectl",
					},
					"kc": {
						Command: "kubectl",
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b"},
			expOutput: heredoc.Doc(`
				EXECUTOR        ENABLED ALIASES RESTARTS STATUS  LAST_RESTART
				botkube/echo    true            0/1      Running 
				botkube/kubectl true    k, kc   0/1      Running`),
		},
		{
			name: "executors and plugins",
			cfg: config.Config{
				Executors: map[string]config.Executors{
					"kubectl": {
						Plugins: config.Plugins{
							"botkube/kubectl": config.Plugin{
								Enabled: true,
							},
						},
					},
					"botkube/helm": {
						Plugins: config.Plugins{
							"botkube/helm": config.Plugin{
								Enabled: true,
							},
						},
					},
					"botkube/echo@v1.0.1-devel": {
						Plugins: config.Plugins{
							"botkube/echo@v1.0.1-devel": config.Plugin{
								Enabled: true,
							},
						},
					},
				},
				Aliases: map[string]config.Alias{
					"h": {
						Command: "helm",
					},
					"e": {
						Command: "echo",
					},
				},
			},
			bindings: []string{"kubectl", "botkube/helm", "botkube/echo@v1.0.1-devel"},
			expOutput: heredoc.Doc(`
				EXECUTOR                  ENABLED ALIASES RESTARTS STATUS  LAST_RESTART
				botkube/echo@v1.0.1-devel true    e       0/1      Running 
				botkube/helm              true    h       0/1      Running 
				botkube/kubectl           true            0/1      Running`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter:    newExecutorTextFilter(""),
				Conversation:      Conversation{ExecutorBindings: tc.bindings},
				PluginHealthStats: plugin.NewHealthStats(1),
			}
			e := NewExecExecutor(loggerx.NewNoop(), tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, msg.BaseBody.CodeBlock)
		})
	}
}
