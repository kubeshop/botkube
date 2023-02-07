package execute

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

// go test -run=TestAliasExecutor_List ./pkg/execute/... -test.update-golden
func TestAliasExecutor_List(t *testing.T) {
	// given
	expContextSections := api.ContextItems{{Text: aliasesForCurrentBindingsMsg}}

	testCases := []struct {
		name     string
		bindings []string
		cfg      config.Config
	}{
		{
			name:     "no bindings",
			bindings: []string{},
			cfg:      fixAliasCfg(),
		},
		{
			name:     "kubectl",
			bindings: []string{"binding1"},
			cfg:      fixAliasCfg(),
		},
		{
			name:     "three bindings",
			bindings: []string{"binding1", "binding2", "plugins"},
			cfg:      fixAliasCfg(),
		},
		{
			name:     "three bindings with builtin cmds",
			bindings: []string{"binding1", "binding2", "plugins"},
			cfg:      fixAliasCfgWithBuiltin(),
		},
		{
			name:     "just builtin cmds",
			bindings: []string{},
			cfg:      fixAliasCfgWithBuiltin(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmdCtx := CommandContext{
				ExecutorFilter: newExecutorTextFilter(""),
				Conversation:   Conversation{ExecutorBindings: tc.bindings},
			}
			e := NewAliasExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, tc.cfg)
			msg, err := e.List(context.Background(), cmdCtx)
			require.NoError(t, err)
			require.Len(t, msg.Sections, 1)
			golden.Assert(t, msg.Sections[0].Body.CodeBlock, fmt.Sprintf("%s.golden.txt", t.Name()))
			assert.Equal(t, expContextSections, msg.Sections[0].Context)
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
				Plugins: config.Plugins{
					"gh": config.Plugin{
						Enabled: false,
					},
				},
			},
			"binding2": {
				Kubectl: config.Kubectl{
					Enabled: false,
				},
				Plugins: config.Plugins{
					"gh": config.Plugin{
						Enabled: true,
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
