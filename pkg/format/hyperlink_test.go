package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/format"
)

func TestRemoveHyperlinks(t *testing.T) {
	type test struct {
		input    string
		expected string
	}

	tests := []test{
		{
			input:    "get <http://prometheuses.monitoring.coreos.com|prometheuses.monitoring.coreos.com> --cluster-name <http://xyz.alpha-sense.org|xyz.alpha-sense.org>",
			expected: "get prometheuses.monitoring.coreos.com --cluster-name xyz.alpha-sense.org",
		},
		{
			input:    "get <http://prometheuses.monitoring.coreos.com|prometheuses.monitoring.coreos.com>",
			expected: "get prometheuses.monitoring.coreos.com",
		},
		{
			input:    "get <https://prometheuses.monitoring.coreos.com|prometheuses.monitoring.coreos.com>",
			expected: "get prometheuses.monitoring.coreos.com",
		},
		{
			input:    "get <https://prometheuses.monitoring.coreos.com>",
			expected: "get https://prometheuses.monitoring.coreos.com",
		},
		{
			input:    "get pods --cluster-name <http://xyz.alpha-sense.org|xyz.alpha-sense.org>",
			expected: "get pods --cluster-name xyz.alpha-sense.org",
		},
		{
			input:    "get pods -n=default",
			expected: "get pods -n=default",
		},
		{
			input:    "get pods",
			expected: "get pods",
		},
	}

	for _, ts := range tests {
		got := format.RemoveHyperlinks(ts.input)
		if got != ts.expected {
			t.Errorf("expected: %v, got: %v", ts.expected, got)
		}
		assert.Equal(t, ts.expected, got)
	}
}
