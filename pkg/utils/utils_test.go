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
