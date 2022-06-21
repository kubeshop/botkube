// Copyright (c) 2020 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package utils

import (
	"testing"
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
