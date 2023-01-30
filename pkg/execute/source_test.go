package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/ptr"
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
						Kubernetes: config.KubernetesSource{
							Recommendations: config.Recommendations{
								Pod: config.PodRecommendations{
									NoLatestImageTag: ptr.Bool(true),
								},
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
				SOURCE       ENABLED
				botkube/helm true
				foo          true
				foo/bar      false
				kubernetes   true
				repo/bar     true`),
		},
		{
			name: "duplicate sources",
			cfg: config.Config{
				Sources: map[string]config.Sources{
					"kubectl-team-a": {
						DisplayName: "kubectl-team-a",
						Kubernetes: config.KubernetesSource{
							Recommendations: config.Recommendations{
								Pod: config.PodRecommendations{
									NoLatestImageTag: ptr.Bool(true),
								},
							},
						},
					},
					"kubectl-team-b": {
						DisplayName: "kubectl-team-b",
						Kubernetes: config.KubernetesSource{
							Recommendations: config.Recommendations{
								Pod: config.PodRecommendations{
									LabelsSet: ptr.Bool(true),
								},
							},
						},
					},
					"plugins": {
						DisplayName: "plugin-a",
						Plugins: config.PluginsMap{
							"plugin-a": {
								Enabled: true,
							},
						},
					},
				},
			},
			bindings: []string{"kubectl-team-a", "kubectl-team-b", "plugins"},
			expOutput: heredoc.Doc(`
				SOURCE     ENABLED
				kubernetes true
				plugin-a   true`),
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
