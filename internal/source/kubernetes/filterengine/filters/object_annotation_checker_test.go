package filters

import (
	"testing"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestIsObjectNotifDisabled(t *testing.T) {
	tests := map[string]struct {
		annotation metaV1.ObjectMeta
		expected   bool
	}{
		`Empty ObjectMeta`:                 {metaV1.ObjectMeta{}, false},
		`ObjectMeta with some annotations`: {metaV1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}, false},
		`ObjectMeta with disable false`:    {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/disable": "false"}}, false},
		`ObjectMeta with disable true`:     {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/disable": "true"}}, true},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			f := NewObjectAnnotationChecker(loggerx.NewNoop(), nil, nil)

			if actual := f.isObjectNotifDisabled(test.annotation); actual != test.expected {
				t.Errorf("expected: %+v != actual: %+v\n", test.expected, actual)
			}
		})
	}
}
