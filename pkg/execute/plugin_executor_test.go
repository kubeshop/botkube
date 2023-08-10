package execute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginExecutor_GetCommandPrefix(t *testing.T) {
	testCases := []struct {
		Name     string
		In       []string
		Expected string
	}{
		{
			Name:     "Single command",
			In:       []string{"kubectl"},
			Expected: "kubectl",
		},
		{
			Name:     "Single verb",
			In:       []string{"kubectl", "get"},
			Expected: "kubectl get",
		},
		{
			Name:     "Multiple args",
			In:       []string{"kubectl", "get", "pods", "-n", "default"},
			Expected: "kubectl get",
		},
		{
			Name:     "Multi-word verb",
			In:       []string{"doctor", "can I ask a question"},
			Expected: "doctor {multi-word arg}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p := &PluginExecutor{}

			// when
			out := p.GetCommandPrefix(tc.In)

			// then
			assert.Equal(t, tc.Expected, out)
		})
	}
}
