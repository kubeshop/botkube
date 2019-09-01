package filters

import (
	"testing"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsObjectNotifDisabled(t *testing.T) {
	tests := map[string]struct {
		annotaion metaV1.ObjectMeta
		expected  bool
	}{
		`Empty ObjectMeta`:                 {metaV1.ObjectMeta{}, false},
		`ObjectMeta with some annotations`: {metaV1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}, false},
		`ObjectMeta with disable false`:    {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/disable": "false"}}, false},
		`ObjectMeta with disable true`:     {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/disable": "true"}}, true},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			if actual := isObjectNotifDisabled(test.annotaion); actual != test.expected {
				t.Errorf("expected: %+v != actual: %+v\n", test.expected, actual)
			}
		})
	}
}

func TestReconfigureChannel(t *testing.T) {
	tests := map[string]struct {
		objectMeta      metaV1.ObjectMeta
		expectedChannel string
		expectedBool    bool
	}{
		`Empty ObjectMeta`:                    {metaV1.ObjectMeta{}, "", false},
		`ObjectMeta with some annotations`:    {metaV1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}, "", false},
		`ObjectMeta with channel ""`:          {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/channel": ""}}, "", false},
		`ObjectMeta with channel foo-channel`: {metaV1.ObjectMeta{Annotations: map[string]string{"botkube.io/channel": "foo-channel"}}, "foo-channel", true},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			if actualChannel, actualBool := reconfigureChannel(test.objectMeta); actualBool != test.expectedBool {
				if actualChannel != test.expectedChannel {
					t.Errorf("expected: %+v != actual: %+v\n", test.expectedChannel, actualChannel)
				}
			}
		})
	}
}
