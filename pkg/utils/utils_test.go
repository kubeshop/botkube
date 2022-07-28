package utils

import (
	"testing"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestGetClusterNameFromKubectlCmd(t *testing.T) {
	type test struct {
		input    string
		expected string
	}

	tests := []test{
		{input: "get pods --cluster-name=minikube", expected: "minikube"},
		{input: "--cluster-name minikube1", expected: "minikube1"},
		{input: "--cluster-name minikube2 -n default", expected: "minikube2"},
		{input: "--cluster-name minikube -n=default", expected: "minikube"},
		{input: "--cluster-name", expected: ""},
		{input: "--cluster-name ", expected: ""},
		{input: "--cluster-name=", expected: ""},
		{input: "", expected: ""},
		{input: "--cluster-nameminikube1", expected: ""},
	}

	for _, ts := range tests {
		got := GetClusterNameFromKubectlCmd(ts.input)
		if got != ts.expected {
			t.Errorf("expected: %v, got: %v", ts.expected, got)
		}
	}
}

func TestContains(t *testing.T) {
	var containsValue = "default"
	var notContainsValue = "demo"
	var commands = []string{
		"get",
		"pods",
		"-n",
		"default",
	}
	expected := true
	got := Contains(commands, containsValue)
	if got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
	expected = false
	got = Contains(commands, notContainsValue)
	if got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestRemoveHypelink(t *testing.T) {
	type test struct {
		input    string
		expected string
	}

	tests := []test{
		{input: "get <http://prometheuses.monitoring.coreos.com|prometheuses.monitoring.coreos.com> --cluster-name <http://xyz.alpha-sense.org|xyz.alpha-sense.org>",
			expected: "get prometheuses.monitoring.coreos.com --cluster-name xyz.alpha-sense.org"},
		{input: "get <http://prometheuses.monitoring.coreos.com|prometheuses.monitoring.coreos.com>",
			expected: "get prometheuses.monitoring.coreos.com"},
		{input: "get pods --cluster-name <http://xyz.alpha-sense.org|xyz.alpha-sense.org>",
			expected: "get pods --cluster-name xyz.alpha-sense.org"},
		{input: "get pods -n=default",
			expected: "get pods -n=default"},
		{input: "get pods",
			expected: "get pods"},
	}

	for _, ts := range tests {
		got := RemoveHyperlink(ts.input)
		if got != ts.expected {
			t.Errorf("expected: %v, got: %v", ts.expected, got)
		}
	}
}

func TestIsNamespaceAllowed(t *testing.T) {
	tests := map[string]struct {
		nsConfig  config.Namespaces
		givenNs   string
		isAllowed bool
	}{
		"should watch all except ignored onces": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo", "abc"}},
			givenNs:   "demo",
			isAllowed: false,
		},
		"should watch all when ignore has empty items only": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: []string{""}},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should watch all when ignore is a nil slice": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: nil},
			givenNs:   "demo",
			isAllowed: true,
		},
		"should ignore matched by regex": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: []string{"my-*"}},
			givenNs:   "my-ns",
			isAllowed: false,
		},
		"should ignore matched by regexp even if exact name is mentioned too": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo", "ignored-*-ns"}},
			givenNs:   "ignored-42-ns",
			isAllowed: false,
		},
		"should watch all if regexp is not matching given namespace": {
			nsConfig:  config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo-*"}},
			givenNs:   "demo",
			isAllowed: true,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual := IsNamespaceAllowed(test.nsConfig, test.givenNs)
			if actual != test.isAllowed {
				t.Errorf("expected: %v != actual: %v\n", test.isAllowed, actual)
			}
		})
	}
}
