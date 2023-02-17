package k8sutil_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/k8sutil"
)

// Object mocks kubernetes objects
type Object struct {
	Spec   Spec   `json:"spec"`
	Status Status `json:"status"`
	Data   Data   `json:"data"`
	Rules  Rules  `json:"rules"`
	Other  Other  `json:"other"`
}

// Other mocks fields like MetaData, Status etc in kubernetes objects
type Other struct {
	Foo         string            `json:"foo"`
	Annotations map[string]string `json:"annotations"`
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

// Event mocks ObjectData field in kubernetes object like configmap
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
		old                Object
		new                Object
		update             config.UpdateSetting
		expected           ExpectedDiff
		expectedErrMessage string
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
			old:                Object{Spec: Spec{Containers: []Container{{Image: "nginx:1.14"}}}, Other: Other{Foo: "bar"}},
			new:                Object{Spec: Spec{Containers: []Container{{Image: "nginx:1.14"}}}, Other: Other{Foo: "boo"}},
			update:             config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expectedErrMessage: "while finding value from jsonpath: \"metadata.name\", object: {Spec:{Port:0 Containers:[{Image:nginx:1.14}]} Status:{Replicas:0} Event:{Properties:} Rules:{Verbs:} Other:{Foo:bar Annotations:map[]}}: metadata is not found",
		},
		`Annotations changed`: {
			old:    Object{Other: Other{Annotations: map[string]string{"app.kubernetes.io/version": "1"}}},
			new:    Object{Other: Other{Annotations: map[string]string{"app.kubernetes.io/version": "2"}}},
			update: config.UpdateSetting{Fields: []string{`other.annotations.app\.kubernetes\.io\/version`}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: `other.annotations.app\.kubernetes\.io\/version`,
				X:    "1",
				Y:    "2",
			},
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
			old:                Object{Status: Status{Replicas: 1}, Other: Other{Foo: "bar"}},
			new:                Object{Status: Status{Replicas: 1}, Other: Other{Foo: "boo"}},
			update:             config.UpdateSetting{Fields: []string{"metadata.labels"}, IncludeDiff: true},
			expectedErrMessage: "while finding value from jsonpath: \"metadata.labels\", object: {Spec:{Port:0 Containers:[]} Status:{Replicas:1} Event:{Properties:} Rules:{Verbs:} Other:{Foo:bar Annotations:map[]}}: metadata is not found",
		},
		`Event Diff`: {
			old:    Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "bar"}},
			new:    Object{Data: Data{Properties: "color: red"}, Other: Other{Foo: "bar"}},
			update: config.UpdateSetting{Fields: []string{"data.properties"}, IncludeDiff: true},
			expected: ExpectedDiff{
				Path: "data.properties",
				X:    "color: blue",
				Y:    "color: red",
			},
		},
		`Non Event Diff`: {
			old:                Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "bar"}},
			new:                Object{Data: Data{Properties: "color: blue"}, Other: Other{Foo: "boo"}},
			update:             config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expectedErrMessage: "while finding value from jsonpath: \"metadata.name\", object: {Spec:{Port:0 Containers:[]} Status:{Replicas:0} Event:{Properties:color: blue} Rules:{Verbs:} Other:{Foo:bar Annotations:map[]}}: metadata is not found",
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
			old:                Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "bar"}},
			new:                Object{Rules: Rules{Verbs: "list"}, Other: Other{Foo: "boo"}},
			update:             config.UpdateSetting{Fields: []string{"metadata.name"}, IncludeDiff: true},
			expectedErrMessage: "while finding value from jsonpath: \"metadata.name\", object: {Spec:{Port:0 Containers:[]} Status:{Replicas:0} Event:{Properties:} Rules:{Verbs:list} Other:{Foo:bar Annotations:map[]}}: metadata is not found",
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual, err := k8sutil.Diff(test.old, test.new, test.update)

			if test.expectedErrMessage != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectedErrMessage, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected.MockDiff(), actual)
		})
	}
}

// MockDiff mocks diff.Diff
func (e *ExpectedDiff) MockDiff() string {
	if e.Path == "" {
		return ""
	}
	return fmt.Sprintf("%+v:\n\t-: %+v\n\t+: %+v\n", e.Path, e.X, e.Y)
}
