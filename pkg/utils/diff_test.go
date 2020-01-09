package utils

import (
	"fmt"
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
)

// Object mocks kubernetes objects
type Object struct {
	Spec   Spec   `json:"spec"`
	Status Status `json:"status"`
	Data   Data   `json:"data"`
	Rules  Rules  `json:"rules"`
	Other  Other  `json:"other"`
}

// Other mocks fileds like MetaData, Status etc in kubernetes objects
type Other struct {
	Foo string `json:"foo"`
}

// Spec mocks ObjectSpec field in kubernetes object
type Spec struct {
	Port       int         `json:"port"`
	Containers []Container `json:"containers"`
}

// Container mocks ObjectSpec.Container field in kubernetes object
type Container struct {
	Image string `json:"image"`
}

// Status mocks ObjectStatus field in kubernetes object
type Status struct {
	Replicas int `json:"replicas"`
}

// Data mocks ObjectData field in kubernetes object like configmap
type Data struct {
	Properties string `json:"properties"`
}

// Rules mocks ObjectRules field in kubernetes object
type Rules struct {
	Verbs string `json:"verbs"`
}

// ExpectedDiff struct to generate expected diff
type ExpectedDiff struct {
	Path string
	X    string
	Y    string
}

func TestDiff(t *testing.T) {
	tests := map[string]struct {
		old      Object
		new      Object
		update   config.UpdateSetting
		expected ExpectedDiff
	}{
		`Spec Diff`: {
			old:    Object{Spec: Spec{Containers: []Container{{Image: "nginx:1.14"}}}, Other: Other{Foo: "bar"}},
			new:    Object{Spec: Spec{Containers: []Container{{Image: "nginx:latest"}}}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"spec.containers[*].image"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "spec.containers[*].image",
				X:    "nginx:1.14",
				Y:    "nginx:latest",
			},
		},
		`Non Spec Diff`: {
			old:      Object{Spec: Spec{Containers: []Container{{Image: "nginx:1.14"}}}, Other: Other{Foo: "bar"}},
			new:      Object{Spec: Spec{Containers: []Container{{Image: "nginx:1.14"}}}, Other: Other{Foo: "boo"}},
			update:   config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expected: ExpectedDiff{},
		},
		`Status Diff`: {
			old:    Object{Status: Status{Replicas: 1}, Other: Other{Foo: "bar"}},
			new:    Object{Status: Status{Replicas: 2}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"status.replicas"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "status.replicas",
				X:    "1",
				Y:    "2",
			},
		},
		`Non Status Diff`: {
			old:      Object{Status: Status{Replicas: 1}, Other: Other{Foo: "bar"}},
			new:      Object{Status: Status{Replicas: 1}, Other: Other{Foo: "boo"}},
			update:   config.UpdateSetting{Fields: []string{"metadata.labels"}, IncludeDiff: true},
			expected: ExpectedDiff{},
		},
		`Data Diff`: {
			old:    Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "bar"}},
			new:    Object{Data: Data{Properties: "color: red"}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"data.properties"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "data.properties",
				X:    "color: blue",
				Y:    "color: red",
			},
		},
		`Non Data Diff`: {
			old:      Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "bar"}},
			new:      Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "boo"}},
			update:   config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expected: ExpectedDiff{},
		},
		`Rules Diff`: {
			old:    Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "bar"}},
			new:    Object{Rules: Rules{Verbs: "watch"}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"rules.verbs"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "rules.verbs",
				X:    "list",
				Y:    "watch",
			},
		},
		`Non Rules Diff`: {
			old:      Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "bar"}},
			new:      Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "boo"}},
			update:   config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expected: ExpectedDiff{},
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			if actual := Diff(test.old, test.new, test.update); actual != test.expected.MockDiff() {
				t.Errorf("expected: %+v != actual: %+v\n", test.expected.MockDiff(), actual)
			}
		})
	}
}

// MockDiff mocks utils.Diff
func (e *ExpectedDiff) MockDiff() string {
	if e.Path == "" {
		return ""
	}
	return fmt.Sprintf("%+v:\n\t-: %+v\n\t+: %+v\n", e.Path, e.X, e.Y)
}
