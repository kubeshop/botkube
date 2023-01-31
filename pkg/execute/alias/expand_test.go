package alias_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/alias"
)

func TestExpand(t *testing.T) {
	// given
	aliases := config.Aliases{
		"k": {
			Command: "kubectl",
		},
		"kc": {
			Command: "kubectl",
		},
		"kgp": {
			Command: "kubectl get pods",
		},
	}
	testCases := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "Single alone alias",
			Input:    "k",
			Expected: "kubectl",
		},
		{
			Name:     "Single letter with additional args",
			Input:    "k get pods -n botkube",
			Expected: "kubectl get pods -n botkube",
		},
		{
			Name:     "Another alias with additional args",
			Input:    "kc -n botkube get pods --filter=boo",
			Expected: "kubectl -n botkube get pods --filter=boo",
		},
		{
			Name:     "Longer alone alias",
			Input:    "kgp",
			Expected: "kubectl get pods",
		},
		{
			Name:     "Longer alias with additional args",
			Input:    "kgp -A",
			Expected: "kubectl get pods -A",
		},
		{
			Name:     "No alias expansion without space",
			Input:    "kc-kubectl get pods kc k",
			Expected: "kc-kubectl get pods kc k",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			actual := alias.ExpandPrefix(tc.Input, aliases)

			// then
			assert.Equal(t, tc.Expected, actual)
		})
	}
}
