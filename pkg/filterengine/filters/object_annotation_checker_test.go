// Copyright (c) 2019 InfraCloud Technologies
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
