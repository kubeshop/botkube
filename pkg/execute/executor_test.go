package execute

import (
	"strings"
	"testing"

	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/require"
)

func TestRemoveBotkubeRelatedFlags(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          string
		ExpectedResult []string
	}{
		{
			Name:           "No input",
			Input:          "@botkube help",
			ExpectedResult: []string{"@botkube", "help"},
		},
		{
			Name:           "Equals sign",
			Input:          "@botkube help --cluster-name=foo",
			ExpectedResult: []string{"@botkube", "help"},
		},
		{
			Name:           "Whitespace",
			Input:          "@botkube help --cluster-name foo",
			ExpectedResult: []string{"@botkube", "help"},
		},
		{
			Name:           "Combination",
			Input:          "@botkube help --cluster-name foo1 --cluster-name=foo2",
			ExpectedResult: []string{"@botkube", "help"},
		},
		{
			Name:           "Kubectl",
			Input:          "@botkube kc get po --cluster-name foo -n default",
			ExpectedResult: []string{"@botkube", "kc", "get", "po", "-n", "default"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			args, err := shellwords.Parse(strings.TrimSpace(tc.Input))
			require.NoError(t, err)
			result, err := removeBotkubeRelatedFlags(args)
			require.NoError(t, err)
			require.ElementsMatch(t, result, tc.ExpectedResult)
		})
	}
}
