package execute

import (
	"context"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestAliasExecutor_List(t *testing.T) {
	// given
	expSections := []interactive.Section{{Context: []interactive.ContextItem{{Text: aliasesForCurrentBindingsMsg}}}}
	cfg := fixAliasCfg()
	testCases := []struct {
		name     string
		bindings []string

		expOutput string
	}{
		{
			name:     "no bindings",
			bindings: []string{},
			expOutput: heredoc.Doc(`
			  No aliases found for current conversation.`),
		},
		{
			name:     "kubectl",
			bindings: []string{"binding1"},
			expOutput: heredoc.Doc(`
			  ALIAS COMMAND                    DISPLAY NAME
			  k     kubectl                    k alias
			  kb    kubectl -n botkube         kubectl for botkube ns
			  kc    kubectl                    
			  kcn   kubectl -n ns              
			  kgp   kubectl get pods           
			  kk    kubectl                    
			  kv    kubectl version --filter=3 version with filter`),
		},
		{
			name:     "three bindings",
			bindings: []string{"binding1", "binding2", "plugins"},
			expOutput: heredoc.Doc(`
			  ALIAS COMMAND      DISPLAY NAME
			  g     gh verb -V   GH verb
			  h     helm         
			  hv    helm version Helm ver`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter: newExecutorTextFilter(""),
				Conversation:   Conversation{ExecutorBindings: tc.bindings},
			}
			e := NewAliasExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, msg.Body.CodeBlock)
			assert.Equal(t, expSections, msg.Sections)
		})
	}
}

func fixAliasCfg() config.Config {
	return config.Config{
		Executors: map[string]config.Executors{
			"binding1": {
				Kubectl: config.Kubectl{
					Enabled: true,
				},
				Plugins: config.PluginsMap{
					"gh": config.Plugin{
						Enabled: false,
					},
				},
			},
			"binding2": {
				Kubectl: config.Kubectl{
					Enabled: false,
				},
				Plugins: config.PluginsMap{
					"gh": config.Plugin{
						Enabled: true,
					},
				},
			},
			"plugins": {
				Plugins: config.PluginsMap{
					"botkube/helm": config.Plugin{
						Enabled: true,
					},
					"botkube/echo@v1.0.1-devel": config.Plugin{
						Enabled: true,
					},
				},
			},
			"other": {
				Plugins: config.PluginsMap{
					"botkube/other@v1.0.1-devel": config.Plugin{
						Enabled: true,
					},
				},
			},
		},
		Aliases: map[string]config.Alias{
			"k": {
				Command:     "kubectl",
				DisplayName: "k alias",
			},
			"kc": {
				Command: "kubectl",
			},
			"kb": {
				Command:     "kubectl -n botkube",
				DisplayName: "kubectl for botkube ns",
			},
			"kv": {
				Command:     "kubectl version --filter=3",
				DisplayName: "version with filter",
			},
			"kdiff": {
				Command: "kubectldiff erent command",
			},
			"kcn": {
				Command: "kubectl -n ns",
			},
			"kk": {
				Command: "kubectl",
			},
			"kgp": {
				Command: "kubectl get pods",
			},
			"h": {
				Command: "helm",
			},
			"hv": {
				Command:     "helm version",
				DisplayName: "Helm ver",
			},
			"o": {
				Command: "other",
			},
			"op": {
				Command: "other --param=1",
			},
			"g": {
				Command:     "gh verb -V",
				DisplayName: "GH verb",
			},
		},
	}
}
