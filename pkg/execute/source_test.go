package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/plugin"
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
						Plugins: map[string]config.Plugin{
							"kubernetes": {
								Enabled: true,
							},
						},
					},
					"kubectl-team-b": {
						DisplayName: "kubectl-team-b",
						Plugins: map[string]config.Plugin{
							"foo": {
								Enabled: true,
							},
							"foo/bar": {
								Enabled: false,
							},
							"repo/bar": {
								Enabled: true,
							},
							"botkube/helm": {
								Enabled: true,
							},
						},
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b"},
			expOutput: heredoc.Doc(`
				SOURCE       ENABLED RESTARTS STATUS  LAST_RESTART
				botkube/helm true    0/1      Running 
				foo          true    0/1      Running 
				foo/bar      false   0/1      Running 
				kubernetes   true    0/1      Running 
				repo/bar     true    0/1      Running`),
		},
		{
			name: "duplicate sources",
			cfg: config.Config{
				Sources: map[string]config.Sources{
					"kubectl-team-a": {
						DisplayName: "kubectl-team-a",
						Plugins: map[string]config.Plugin{
							"kubernetes": {
								Enabled: true,
							},
						},
					},
					"kubectl-team-b": {
						DisplayName: "kubectl-team-b",
						Plugins: map[string]config.Plugin{
							"kubernetes": {
								Enabled: true,
							},
						},
					},
					"plugins": {
						DisplayName: "plugin-a",
						Plugins: config.Plugins{
							"plugin-a": {
								Enabled: true,
							},
						},
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b", "plugins"},
			expOutput: heredoc.Doc(`
				SOURCE     ENABLED RESTARTS STATUS  LAST_RESTART
				kubernetes true    0/1      Running 
				plugin-a   true    0/1      Running`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter:    newExecutorTextFilter(""),
				Conversation:      Conversation{SourceBindings: tc.bindings},
				PluginHealthStats: plugin.NewHealthStats(1),
			}
			e := NewSourceExecutor(loggerx.NewNoop(), tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, msg.BaseBody.CodeBlock)
		})
	}
}
