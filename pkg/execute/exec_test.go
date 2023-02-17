package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
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
				EXECUTOR        ENABLED ALIASES
				botkube/echo    true    
				botkube/kubectl true    k, kc`),
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
				EXECUTOR                  ENABLED ALIASES
				botkube/echo@v1.0.1-devel true    e
				botkube/helm              true    h
				botkube/kubectl           true`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter: newExecutorTextFilter(""),
				Conversation:   Conversation{ExecutorBindings: tc.bindings},
			}
			e := NewExecExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, msg.BaseBody.CodeBlock)
		})
	}
}
