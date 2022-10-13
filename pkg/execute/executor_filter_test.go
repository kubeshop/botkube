package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
)

func TestExecutorEchoFilter_Apply(t *testing.T) {
	var filter executorFilter = newExecutorEchoFilter("")

	text := "Please return this same text."
	assert.Equal(t, text, filter.Apply(text))
}

func TestExecutorTextFilter_Apply(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name: "filter multi line text",
			text: heredoc.Doc(`
NAME                                             READY   STATUS    RESTARTS   AGE
pod/coredns-558bd4d5db-c5gwx                     1/1     Running   0          30m
pod/coredns-558bd4d5db-j5wqt                     1/1     Running   0          30m
pod/etcd-kind-control-plane                      1/1     Running   0          30m
pod/kindnet-hl6zc                                1/1     Running   0          29m
pod/kindnet-tc254                                1/1     Running   0          30m
pod/kindnet-x79x6                                1/1     Running   0          29m

NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
daemonset.apps/kindnet      3         3         3       3            3           <none>                   30m
daemonset.apps/kube-proxy   3         3         3       3            3           kubernetes.io/os=linux   30m`),
			expected: heredoc.Doc(`
pod/etcd-kind-control-plane                      1/1     Running   0          30m
pod/kindnet-hl6zc                                1/1     Running   0          29m
pod/kindnet-tc254                                1/1     Running   0          30m
pod/kindnet-x79x6                                1/1     Running   0          29m
daemonset.apps/kindnet      3         3         3       3            3           <none>                   30m`),
		},
		{
			name:     "filter single line text",
			text:     `pod/etcd-kind-control-plane                      1/1     Running   0          30m`,
			expected: `pod/etcd-kind-control-plane                      1/1     Running   0          30m`,
		},
		{
			name:     "no match filter",
			text:     `pod/etcd-control-plane                      1/1     Running   0          30m`,
			expected: "",
		},
	}

	var txFilter executorFilter = newExecutorTextFilter("kind", "")
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, txFilter.Apply(tc.text))
		})
	}
}

func TestExtractExecutorFilter_NoErrors(t *testing.T) {
	testCases := []struct {
		name          string
		cmd           string
		extractedCmd  string
		text          string
		filterApplied string
		filterActive  bool
	}{
		{
			name:          "extract unquoted text filter at end of command",
			cmd:           `kubectl get po -n kube-system --filter=kind`,
			extractedCmd:  "kubectl get po -n kube-system",
			text:          `etcd-kind-control-plane                      1/1     Running   0          86m`,
			filterApplied: `etcd-kind-control-plane                      1/1     Running   0          86m`,
			filterActive:  true,
		},
		{
			name:          "extract unquoted text filter in the middle of the command",
			cmd:           `kubectl get po  --filter=kind -n kube-system`,
			extractedCmd:  "kubectl get po  -n kube-system",
			text:          `etcd-control-plane                      1/1     Running   0          86m`,
			filterApplied: "",
			filterActive:  true,
		},
		{
			name:          "extract single quoted text filter in the middle of the command",
			cmd:           `kubectl get po  --filter="kind system" -n kube-system`,
			extractedCmd:  "kubectl get po  -n kube-system",
			text:          `etcd-control-plane                      1/1     Running   0          86m`,
			filterApplied: "",
			filterActive:  true,
		},
		{
			name:          "extract double quoted text filter in the middle of the command",
			cmd:           `kubectl get po  --filter="kind" -n kube-system`,
			extractedCmd:  "kubectl get po  -n kube-system",
			text:          `etcd-control-plane                      1/1     Running   0          86m`,
			filterApplied: "",
			filterActive:  true,
		},
		{
			name:          "extract double quoted text filter with extra spaces in the command",
			cmd:           `kubectl get po  --filter      "kind" -n kube-system`,
			extractedCmd:  "kubectl get po  -n kube-system",
			text:          `etcd-control-plane                      1/1     Running   0          86m`,
			filterApplied: "",
			filterActive:  true,
		},
		{
			name:          "extract echo filter from command",
			cmd:           "kubectl get po -n kube-system",
			extractedCmd:  "kubectl get po -n kube-system",
			text:          `etcd-control-plane                      1/1     Running   0          86m`,
			filterApplied: `etcd-control-plane                      1/1     Running   0          86m`,
			filterActive:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := extractExecutorFilter(tc.cmd)
			assert.Nil(t, err)
			assert.Equal(t, tc.extractedCmd, filter.FilteredCommand())
			assert.Equal(t, tc.filterApplied, filter.Apply(tc.text))
			assert.Equal(t, tc.filterActive, filter.IsActive())
		})
	}
}

func TestExtractExecutorFilter_WithErrors(t *testing.T) {
	testCases := []struct {
		name   string
		cmd    string
		errMsg string
	}{
		{
			name:   "raise error when filter value is missing at end of command",
			cmd:    "kubectl get po -n kube-system --filter",
			errMsg: `flag needs an argument`,
		},
		{
			name:   "raise error when filter value is missing in the middle of command",
			cmd:    "kubectl get po --filter -n kube-system",
			errMsg: `flag needs an argument`,
		},
		{
			name:   "raise error when multiple filter flags with values  are used in command",
			cmd:    "kubectl get po --filter hello --filter='world' -n kube-system",
			errMsg: `found more than one filter flag`,
		},
		{
			name:   "raise error when multiple filter flags with no values are used in command",
			cmd:    "kubectl get po --filter --filter -n kube-system",
			errMsg: `flag needs an argument`,
		},
		{
			name:   "raise error when filter flag with equal operator and extra spaces in the command",
			cmd:    `kubectl get po --filter=    "kind" -n kube-system`,
			errMsg: `flag needs an argument`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractExecutorFilter(tc.cmd)
			assert.ErrorContains(t, err, tc.errMsg)
		})
	}
}
