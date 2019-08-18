package utils

import (
	"fmt"
	"testing"
)

// Object mocks kubernetes objects
type Object struct {
	Spec  Spec
	Other Other
}

// Other mocks fileds like MetaData, Status etc in kubernetes objects
type Other struct {
	Foo string
}

// Spec mocks ObjectSpec field in kubernetes object
type Spec struct {
	Port int
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
		expected ExpectedDiff
	}{
		`Spec Diff`: {
			old: Object{Spec: Spec{Port: 81}, Other: Other{Foo: "bar"}},
			new: Object{Spec: Spec{Port: 83}, Other: Other{Foo: "bar"}},
			expected: ExpectedDiff{
				Path: "{utils.Object}.Spec.Port",
				X:    "81",
				Y:    "83",
			},
		},
		`Non Spec Diff`: {
			old:      Object{Spec: Spec{Port: 81}, Other: Other{Foo: "bar"}},
			new:      Object{Spec: Spec{Port: 81}, Other: Other{Foo: "boo"}},
			expected: ExpectedDiff{},
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			if actual := Diff(test.old, test.new); actual != test.expected.MockDiff() {
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
