package execute_test

import (
	"testing"

	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/stretchr/testify/assert"
)

func TestEchoFilter_Apply(t *testing.T) {
	var filter execute.ResultsFilter = execute.NewEchoFilter()

	text := "Please return this same text."
	assert.Equal(t, text, filter.Apply(text))
}

func TestTextFilter_Apply(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name: "filter multi line text",
			text: `NAME                                             READY   STATUS    RESTARTS   AGE
pod/coredns-558bd4d5db-c5gwx                     1/1     Running   0          30m
pod/coredns-558bd4d5db-j5wqt                     1/1     Running   0          30m
pod/etcd-kind-control-plane                      1/1     Running   0          30m
pod/kindnet-hl6zc                                1/1     Running   0          29m
pod/kindnet-tc254                                1/1     Running   0          30m
pod/kindnet-x79x6                                1/1     Running   0          29m

NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
daemonset.apps/kindnet      3         3         3       3            3           <none>                   30m
daemonset.apps/kube-proxy   3         3         3       3            3           kubernetes.io/os=linux   30m`,
			expected: `pod/etcd-kind-control-plane                      1/1     Running   0          30m
pod/kindnet-hl6zc                                1/1     Running   0          29m
pod/kindnet-tc254                                1/1     Running   0          30m
pod/kindnet-x79x6                                1/1     Running   0          29m
daemonset.apps/kindnet      3         3         3       3            3           <none>                   30m`,
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

	filterText := "kind"
	var txFilter execute.ResultsFilter = execute.NewTextFilter(filterText)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, txFilter.Apply(tc.text))
		})
	}
}
