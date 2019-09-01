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
		`include all and ignore few --> watch all except ignored`: {config.Namespaces{Include: []string{"all"}, Ignore: []string{"demo", "abc"}}, "demo", true},
		`include all and ignore is "" --> watch all`:              {config.Namespaces{Include: []string{"all"}, Ignore: []string{""}}, "demo", false},
		`include all and ignore is [] --> watch all`:              {config.Namespaces{Include: []string{"all"}, Ignore: []string{}}, "demo", false},
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
