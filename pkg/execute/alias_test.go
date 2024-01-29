package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestAliasExecutor_List(t *testing.T) {
	// given
	expContextSections := api.ContextItems{{Text: aliasesForCurrentBindingsMsg}}

	testCases := []struct {
		name     string
		bindings []string
		cfg      config.Config

		expOutput string
	}{
		{
			name:     "no bindings",
			bindings: []string{},
			cfg:      fixAliasCfg(),
			expOutput: heredoc.Doc(`
			  No aliases found for current conversation.`),
		},
		{
			name:     "kubectl",
			bindings: []string{"binding1"},
			cfg:      fixAliasCfg(),
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
			cfg:      fixAliasCfg(),
			expOutput: heredoc.Doc(`
			  ALIAS COMMAND                    DISPLAY NAME
			  g     gh verb -V                 GH verb
			  h     helm                       
			  hv    helm version               Helm ver
			  k     kubectl                    k alias
			  kb    kubectl -n botkube         kubectl for botkube ns
			  kc    kubectl                    
			  kcn   kubectl -n ns              
			  kgp   kubectl get pods           
			  kk    kubectl                    
			  kv    kubectl version --filter=3 version with filter`),
		},
		{
			name:     "three bindings with builtin cmds",
			bindings: []string{"binding1", "binding2", "plugins"},
			cfg:      fixAliasCfgWithBuiltin(),
			expOutput: heredoc.Doc(`
			  ALIAS COMMAND                    DISPLAY NAME
			  bkh   help                       Botkube Help
			  g     gh verb -V                 GH verb
			  h     helm                       
			  hv    helm version               Helm ver
			  k     kubectl                    k alias
			  kb    kubectl -n botkube         kubectl for botkube ns
			  kc    kubectl                    
			  kcn   kubectl -n ns              
			  kgp   kubectl get pods           
			  kk    kubectl                    
			  kv    kubectl version --filter=3 version with filter
			  p     ping                       Botkube Ping`),
		},
		{
			name:     "just builtin cmds",
			bindings: []string{},
			cfg:      fixAliasCfgWithBuiltin(),
			expOutput: heredoc.Doc(`
			  ALIAS COMMAND DISPLAY NAME
			  bkh   help    Botkube Help
			  p     ping    Botkube Ping`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter: newExecutorTextFilter(""),
				Conversation:   Conversation{ExecutorBindings: tc.bindings},
			}
			e := NewAliasExecutor(loggerx.NewNoop(), tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			require.Len(t, msg.Sections, 1)
			assert.Equal(t, tc.expOutput, msg.Sections[0].Body.CodeBlock)
			assert.Equal(t, expContextSections, msg.Sections[0].Context)
		})
	}
}

func fixAliasCfg() config.Config {
	return config.Config{
		Executors: map[string]config.Executors{
			"binding1": {
				Plugins: config.Plugins{
					"gh": config.Plugin{
						Enabled: false,
					},
					"botkube/kubectl": config.Plugin{
						Enabled: true,
					},
				},
			},
			"binding2": {
				Plugins: config.Plugins{
					"gh": config.Plugin{
						Enabled: true,
					},
					"botkube/kubectl": config.Plugin{
						Enabled: false,
					},
				},
			},
			"plugins": {
				Plugins: config.Plugins{
					"botkube/helm": config.Plugin{
						Enabled: true,
					},
					"botkube/echo@v1.0.1-devel": config.Plugin{
						Enabled: true,
					},
				},
			},
			"other": {
				Plugins: config.Plugins{
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

func fixAliasCfgWithBuiltin() config.Config {
	cfg := fixAliasCfg()
	cfg.Aliases["p"] = config.Alias{
		Command:     "ping",
		DisplayName: "Botkube Ping",
	}
	cfg.Aliases["bkh"] = config.Alias{
		Command:     "help",
		DisplayName: "Botkube Help",
	}
	return cfg
}
