package alias_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/alias"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

func TestListForExecutor(t *testing.T) {
	// given
	aliases := fixAliasesCfg()
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
			actual := alias.ListExactForExecutor(tc.Input, aliases)

			// then
			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func TestListForExecutorPrefix(t *testing.T) {
	// given
	aliases := fixAliasesCfg()
	testCases := []struct {
		Name     string
		Input    string
		Expected []string
	}{
		{
			Name:     "Multiple aliases",
			Input:    "kubectl",
			Expected: []string{"k", "kb", "kc", "kcn", "kgp", "kk", "kv"},
		},
		{
			Name:     "No alias for plugin",
			Input:    "botkube/echo",
			Expected: nil,
		},
		{
			Name:     "No alias for builtin executor",
			Input:    "echo",
			Expected: nil,
		},
		{
			Name:     "Helm",
			Input:    "botkube/helm",
			Expected: []string{"h", "hv"},
		},
		{
			Name:     "Plugin",
			Input:    "anything/kubectl",
			Expected: []string{"k", "kb", "kc", "kcn", "kgp", "kk", "kv"},
		},
		{
			Name:     "Plugin with version",
			Input:    "anything/kubectl@v1.0.0",
			Expected: []string{"k", "kb", "kc", "kcn", "kgp", "kk", "kv"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			actual := alias.ListForExecutorPrefix(tc.Input, aliases)

			// then
			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func TestListForBuiltinVerbPrefix(t *testing.T) {
	// given
	aliases := config.Aliases{
		"e": {
			Command: "edit",
		},
		"esb": {
			Command: "edit sourcebindings foo,bar",
		},
		"p": {
			Command: "ping",
		},
		"pp": {
			Command: "ping --cluster-name=dev",
		},
	}

	testCases := []struct {
		Name     string
		Input    command.Verb
		Expected []string
	}{
		{
			Name:     "Ping",
			Input:    "ping",
			Expected: []string{"p", "pp"},
		},
		{
			Name:     "edit",
			Input:    "edit",
			Expected: []string{"e", "esb"},
		},
		{
			Name:     "No alias for builtin",
			Input:    "make",
			Expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			actual := alias.ListForBuiltinVerbPrefix(tc.Input, aliases)

			// then
			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func fixAliasesCfg() config.Aliases {
	return config.Aliases{
		"k": {
			Command: "kubectl",
		},
		"kc": {
			Command: "kubectl",
		},
		"kb": {
			Command: "kubectl -n botkube",
		},
		"kv": {
			Command: "kubectl version --filter=3",
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
			Command: "helm version",
		},
	}
}
