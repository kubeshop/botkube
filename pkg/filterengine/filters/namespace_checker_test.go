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

	"github.com/infracloudio/botkube/pkg/config"
)

func TestIsNamespaceIgnored(t *testing.T) {
	tests := map[string]struct {
		Namespaces     config.Namespaces
		eventNamespace string
		expected       bool
	}{
		`include all and ignore few --> watch all except ignored`:                      {config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo", "abc"}}, "demo", true},
		`include all and ignore is "" --> watch all`:                                   {config.Namespaces{Include: []string{"all"}, Ignore: []string{""}}, "demo", false},
		`include all and ignore is [] --> watch all`:                                   {config.Namespaces{Include: []string{"all"}, Ignore: []string{}}, "demo", false},
		`include all and ignore with reqexp --> watch all except matched`:              {config.Namespaces{Include: []string{"all"}, Ignore: []string{"my-*"}}, "my-ns", true},
		`include all and ignore few combined with regexp --> watch all except ignored`: {config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo", "ignored-*-ns"}}, "ignored-42-ns", true},
		`include all and ignore with regexp that doesn't match anything --> watch all`: {config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo-*"}}, "demo", false},
		// utils.AllowedEventKindsMap inherently handles remaining test case
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual := isNamespaceIgnored(test.Namespaces, test.eventNamespace)
			if actual != test.expected {
				t.Errorf("expected: %+v != actual: %+v\n", test.expected, actual)
			}
		})
	}
}
