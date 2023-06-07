package execute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveBotkubeRelatedFlags(t *testing.T) {
	testCases := []struct {
		Name        string
		Input       string
		Cmd         string
		ClusterName string
		Filter      string
	}{
		{
			Name:        "Combination cluster name and filter + quotes",
			Input:       `kubectl get po -A --filter="before after" --cluster-name=cluster-a1`,
			Cmd:         "kubectl get po -A",
			ClusterName: "cluster-a1",
			Filter:      "before after",
		},
		{
			Name:        "No input",
			Input:       "@botkube help",
			Cmd:         "@botkube help",
			ClusterName: "",
			Filter:      "",
		},
		{
			Name:        "Equals sign",
			Input:       "@botkube help --cluster-name=foo",
			Cmd:         "@botkube help",
			ClusterName: "foo",
			Filter:      "",
		},
		{
			Name:        "Whitespace",
			Input:       "@botkube help --cluster-name foo",
			Cmd:         "@botkube help",
			ClusterName: "foo",
			Filter:      "",
		},
		{
			Name:        "Combination",
			Input:       "@botkube help --cluster-name \"foo1\"",
			Cmd:         "@botkube help",
			ClusterName: "foo1",
			Filter:      "",
		},
		{
			Name:        "Kubectl",
			Input:       "@botkube kc get po --cluster-name foo -n default",
			Cmd:         "@botkube kc get po -n default",
			ClusterName: "foo",
			Filter:      "",
		},
		{
			Name:        "Remove empty cluster name with equals sign",
			Input:       "@botkube help --cluster-name=",
			Cmd:         "@botkube help",
			ClusterName: "",
			Filter:      "",
		},
		{
			Name:        "Combination cluster name and filter",
			Input:       "@botkube help --cluster-name \"foo1\" --filter=api",
			Cmd:         "@botkube help",
			ClusterName: "foo1",
			Filter:      "api",
		},
		{
			Name:        "Combination cluster name and filter + quotes",
			Input:       "@botkube help --cluster-name \"foo1\" --filter=\"api\"",
			Cmd:         "@botkube help",
			ClusterName: "foo1",
			Filter:      "api",
		},
		{
			Name:        "Extract double quoted text filter with special characters",
			Input:       `@botkube help --cluster-name="api" --filter="botkube.   . [] *?   ^  ===== /test/"`,
			Cmd:         "@botkube help",
			ClusterName: "api",
			Filter:      "botkube.   . [] *?   ^  ===== /test/",
		},
		{
			Name:        "Extract double quoted text filter with a file path",
			Input:       `@botkube help --cluster-name="api" --filter="=./Users/botkube/somefile.txt [info]"`,
			Cmd:         "@botkube help",
			ClusterName: "api",
			Filter:      "=./Users/botkube/somefile.txt [info]",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p, err := ParseFlags(tc.Input)
			require.NoError(t, err)
			require.Equal(t, tc.Cmd, p.CleanCmd)
			require.Equal(t, tc.ClusterName, p.ClusterName)
			require.Equal(t, tc.Filter, p.Filter)
		})
	}
}

func TestExtractExecutorFilter_WithErrors(t *testing.T) {
	testCases := []struct {
		Name   string
		Cmd    string
		ErrMsg string
	}{
		{
			Name:   "raise error when filter value is missing at end of command",
			Cmd:    "kubectl get po -n kube-system --filter",
			ErrMsg: `flag needs an argument`,
		},
		{
			Name:   "raise error when filter value is missing in the middle of command",
			Cmd:    "kubectl get po --filter -n kube-system",
			ErrMsg: `an argument is missing`,
		},
		{
			Name:   "raise error when multiple filter flags with values  are used in command",
			Cmd:    "kubectl get po --filter hello --filter='world' -n kube-system",
			ErrMsg: `found more than one filter flag`,
		},
		{
			Name:   "raise error when multiple filter flags with no values are used in command",
			Cmd:    "kubectl get po --filter --filter -n kube-system",
			ErrMsg: `an argument is missing`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := ParseFlags(tc.Cmd)
			assert.ErrorContains(t, err, tc.ErrMsg)
		})
	}
}
