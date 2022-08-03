package kubectl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

func TestKubectlMerger(t *testing.T) {
	// given
	tests := []struct {
		name string

		givenBindings       []string
		expectKubectlConfig kubectl.EnabledKubectl
		givenNamespace      string
	}{
		{
			name: "Should collect settings with ignored settings for team-b",
			givenBindings: []string{
				"kubectl-team-a",
				"kubectl-team-b",
				"kubectl-global",
				"kubectl-exec",
			},
			givenNamespace: "team-a",
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get":  {},
					"logs": {},
					"top":  {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
				},
				DefaultNamespace: "team-a",
				RestrictAccess:   true,
			},
		},
		{
			name: "Should collect settings only for 'all' namespace",
			givenBindings: []string{
				"kubectl-team-a",
				"kubectl-team-b",
				"kubectl-global",
				"kubectl-exec",
				"kubectl-all",
			},
			givenNamespace: config.AllNamespaceIndicator,
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"logs":         {},
					"top":          {},
					"cluster-info": {},
				},
				AllowedKubectlResource: map[string]struct{}{},
				DefaultNamespace:       "foo",
				RestrictAccess:         true,
			},
		},
		{
			name: "Should collect only team-a settings",
			givenBindings: []string{
				"kubectl-team-a",
				"kubectl-team-b",
			},
			givenNamespace: "team-a",
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get": {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
				},
				DefaultNamespace: "team-a",
				RestrictAccess:   false,
			},
		},
		{
			name: "Should enable restrict access based on the bindings order",
			givenBindings: []string{
				"kubectl-team-a", // disables restrict
				"kubectl-global", // enables restrict
			},
			givenNamespace: "team-a",
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get":  {},
					"logs": {},
					"top":  {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
				},
				DefaultNamespace: "team-a",
				RestrictAccess:   true,
			},
		},
		{
			name: "Should disable restrict access based on the bindings order",
			givenBindings: []string{
				"kubectl-global", // enables restrict
				"kubectl-team-a", // disables restrict
			},
			givenNamespace: "team-a",
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get":  {},
					"logs": {},
					"top":  {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
				},
				DefaultNamespace: "team-a",
				RestrictAccess:   false,
			},
		},
		{
			name: "Should enable restrict access based on the bindings order",
			givenBindings: []string{
				"kubectl-global", // enables restrict
				"kubectl-team-b", // doesn't specify restrict
			},
			givenNamespace: "team-b",
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get":      {},
					"describe": {},
					"logs":     {},
					"top":      {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
					"pods":        {},
				},
				RestrictAccess: true,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			kubectlMerger := kubectl.NewMerger(fixExecutorsConfig(t))

			// when
			gotKubectlConfig := kubectlMerger.MergeForNamespace(tc.givenBindings, tc.givenNamespace)

			// then
			assert.Equal(t, tc.expectKubectlConfig, gotKubectlConfig)
		})
	}
}
