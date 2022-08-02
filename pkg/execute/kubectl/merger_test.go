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
			name: "Should collect settings with ignore settings for team-b",
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
				RestrictAccess:   false,
			},
		},
		{
			name: "Should collect all settings",
			givenBindings: []string{
				"kubectl-team-a",
				"kubectl-team-b",
				"kubectl-global",
				"kubectl-exec",
			},
			givenNamespace: config.AllNamespaceIndicator,
			expectKubectlConfig: kubectl.EnabledKubectl{
				AllowedKubectlVerb: map[string]struct{}{
					"get":      {},
					"logs":     {},
					"top":      {},
					"describe": {},
				},
				AllowedKubectlResource: map[string]struct{}{
					"deployments": {},
					"pods":        {},
				},
				DefaultNamespace: "team-a",
				RestrictAccess:   false,
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			kubectlMerger := kubectl.NewMerger(fixExecutorsConfig(t))

			// when
			gotKubectlConfig := kubectlMerger.Merge(tc.givenBindings, tc.givenNamespace)

			// then
			assert.Equal(t, tc.expectKubectlConfig, gotKubectlConfig)
		})
	}
}
