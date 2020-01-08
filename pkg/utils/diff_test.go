package utils

import (
	"fmt"
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
)

// Object mocks kubernetes objects
type Object struct {
	Spec   Spec
	Status Status
	Data   Data
	Rules  Rules
	Other  Other
}

// Other mocks fileds like MetaData, Status etc in kubernetes objects
type Other struct {
	Foo string
}

// Spec mocks ObjectSpec field in kubernetes object
type Spec struct {
	Port       int
	Containers []Container
}

// Container mocks ObjectSpec.Container field in kubernetes object
type Container struct {
	Image string
}

// Status mocks ObjectStatus field in kubernetes object
type Status struct {
	Replicas int
}

// Data mocks ObjectData field in kubernetes object like configmap
type Data struct {
	Properties string
}

// Rules mocks ObjectRules field in kubernetes object
type Rules struct {
	Verbs string
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
			update: config.UpdateSetting{Fields: []string{"Spec.Containers[*].Image"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "Spec.Containers[*].Image",
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
			update: config.UpdateSetting{Fields: []string{"Status.Replicas"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "Status.Replicas",
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
			old:    Object{Data: Data{Properties: "Color: blue"}, Other: Other{Foo: "bar"}},
			new:    Object{Data: Data{Properties: "Color: red"}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"Data.Properties"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "Data.Properties",
				X:    "Color: blue",
				Y:    "Color: red",
			},
		},
		`Non Data Diff`: {
			old:      Object{Data: Data{Properties: "Color: blue"}, Other: Other{Foo: "bar"}},
			new:      Object{Data: Data{Properties: "Color: blue"}, Other: Other{Foo: "boo"}},
			update:   config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expected: ExpectedDiff{},
		},
		`Rules Diff`: {
			old:    Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "bar"}},
			new:    Object{Rules: Rules{Verbs: "watch"}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"Rules.Verbs"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "Rules.Verbs",
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
