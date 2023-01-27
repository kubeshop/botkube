package alias_test

import (
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/alias"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestListForExecutor(t *testing.T) {
	// given
	aliases := config.Aliases{
		"k": {
			Command: "kubectl",
		},
		"kc": {
			Command: "kubectl",
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
			Command: "helm version",
		},
	}
	testCases := []struct {
		Name     string
		Input    string
		Expected []string
	}{
		{
			Name:     "Multiple aliases",
			Input:    "kubectl",
			Expected: []string{"k", "kc", "kk"},
		},
		{
			Name:     "No alias for plugin",
			Input:    "botkube/echo",
			Expected: nil,
		},
		{
			Name:     "No alias for builtin",
			Input:    "echo",
			Expected: nil,
		},
		{
			Name:     "Helm",
			Input:    "botkube/helm",
			Expected: []string{"h"},
		},
		{
			Name:     "Plugin",
			Input:    "anything/kubectl",
			Expected: []string{"k", "kc", "kk"},
		},
		{
			Name:     "Plugin with version",
			Input:    "anything/kubectl@v1.0.0",
			Expected: []string{"k", "kc", "kk"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			actual := alias.ListForExecutor(tc.Input, aliases)

			// then
			assert.Equal(t, tc.Expected, actual)
		})
	}
}
