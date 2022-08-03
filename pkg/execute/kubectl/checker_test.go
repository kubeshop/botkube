package kubectl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

func TestKubectlCheckerIsResourceAllowedInNs(t *testing.T) {
	tests := []struct {
		name string

		namespace string
		bindings  []string
		resource  string
		variantFn kubectl.ResourceVariantsFunc

		expIsAllowed bool
	}{
		{
			name:      "Should allow deployments",
			namespace: "team-a",
			resource:  "deployments",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},

			expIsAllowed: true,
		},
		{
			name:      "Should allow deploy variant",
			namespace: "team-a",
			resource:  "deploy",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},
			variantFn: func(resource string) []string {
				if resource == "deploy" {
					return []string{"deployments"}
				}
				return nil
			},

			expIsAllowed: true,
		},
		{
			name:      "Should not allow pods",
			namespace: "team-a",
			resource:  "pods",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},
			variantFn: func(resource string) []string {
				if resource == "deploy" {
					return []string{"deployments"}
				}
				return nil
			},

			expIsAllowed: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := kubectl.NewMerger(fixExecutorsConfig(t)).MergeForNamespace(tc.bindings, tc.namespace)
			checker := kubectl.NewChecker(tc.variantFn)

			// when
			gotIsAllowed := checker.IsResourceAllowedInNs(config, tc.resource)

			// then
			assert.Equal(t, tc.expIsAllowed, gotIsAllowed)
		})
	}
}

func TestKubectlCheckerIsVerbAllowedInNs(t *testing.T) {
	tests := []struct {
		name string

		namespace string
		bindings  []string
		verb      string

		expIsAllowed bool
	}{
		{
			name:      "Should allow get from team-a settings",
			namespace: "team-a",
			verb:      "get",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},

			expIsAllowed: true,
		},
		{
			name:      "Should allow logs taken from global settings",
			namespace: "team-a",
			verb:      "logs",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},

			expIsAllowed: true,
		},
		{
			name:      "Should not allow pods",
			namespace: "team-a",
			verb:      "exec",
			bindings: []string{
				"kubectl-team-a",
				"kubectl-global",
			},

			expIsAllowed: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := kubectl.NewMerger(fixExecutorsConfig(t)).MergeForNamespace(tc.bindings, tc.namespace)
			checker := kubectl.NewChecker(nil)

			// when
			gotIsAllowed := checker.IsVerbAllowedInNs(config, tc.verb)

			// then
			assert.Equal(t, tc.expIsAllowed, gotIsAllowed)
		})
	}
}
