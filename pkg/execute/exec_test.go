package execute

import (
	"context"
	"fmt"
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
						Kubectl: config.Kubectl{
							Enabled: true,
						},
					},
					"kubectl-team-b": {
						Kubectl: config.Kubectl{
							Enabled: false,
						},
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b"},
			expOutput: heredoc.Doc(`
				EXECUTOR       ENABLED
				kubectl-team-a true
				kubectl-team-b false`),
		},
		{
			name: "executors and plugins",
			cfg: config.Config{
				Executors: map[string]config.Executors{
					"kubectl-exec-cmd": {
						Kubectl: config.Kubectl{
							Enabled: false,
						},
					},
					"kubectl-read-only": {
						Kubectl: config.Kubectl{
							Enabled: true,
						},
					},

					"kubectl-wait-cmd": {
						Kubectl: config.Kubectl{
							Enabled: true,
						},
					},
					"botkube/helm": {
						Plugins: config.PluginsExecutors{
							"botkube/helm": config.PluginExecutor{
								Enabled: true,
							},
						},
					},
					"botkube/echo@v1.0.1-devel": {
						Plugins: config.PluginsExecutors{
							"botkube/echo@v1.0.1-devel": config.PluginExecutor{
								Enabled: true,
							},
						},
					},
				},
			},
			bindings: []string{"kubectl-exec-cmd", "kubectl-read-only", "kubectl-wait-cmd", "botkube/helm", "botkube/echo@v1.0.1-devel"},
			expOutput: heredoc.Doc(`
				EXECUTOR                  ENABLED
				botkube/echo@v1.0.1-devel true
				botkube/helm              true
				kubectl-exec-cmd          false
				kubectl-read-only         true
				kubectl-wait-cmd          true`),
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
			fmt.Println(msg.Body.CodeBlock)
			assert.Equal(t, tc.expOutput, msg.Body.CodeBlock)
		})
	}
}
