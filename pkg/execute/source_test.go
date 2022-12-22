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

func TestSourceExecutor(t *testing.T) {
	testCases := []struct {
		name     string
		cfg      config.Config
		bindings []string

		expOutput string
	}{
		{
			name: "two sources",
			cfg: config.Config{
				Sources: map[string]config.Sources{
					"kubectl-team-a": {
						DisplayName: "kubectl-team-a",
					},
					"kubectl-team-b": {
						DisplayName: "kubectl-team-b",
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b"},
			expOutput: heredoc.Doc(`SOURCE         ENABLED DISPLAY NAME
            kubectl-team-a true    kubectl-team-a
            kubectl-team-b true    kubectl-team-b`),
		},
		{
			name: "two sources with plugin",
			cfg: config.Config{
				Sources: map[string]config.Sources{
					"kubectl-team-a": {
						DisplayName: "kubectl-team-a",
					},
					"kubectl-team-b": {
						DisplayName: "kubectl-team-b",
					},
					"plugin-a": {
						DisplayName: "plugin-a",
						Plugins: config.PluginsExecutors{
							"plugin-a": {
								Enabled: true,
							},
						},
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b", "plugin-a"},
			expOutput: heredoc.Doc(`SOURCE         ENABLED DISPLAY NAME
            kubectl-team-a true    kubectl-team-a
            kubectl-team-b true    kubectl-team-b
            plugin-a       true    plugin-a`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter: newExecutorTextFilter(""),
				Conversation:   Conversation{SourceBindings: tc.bindings},
			}
			e := NewSourceExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, msg.Body.CodeBlock)
		})
	}
}
