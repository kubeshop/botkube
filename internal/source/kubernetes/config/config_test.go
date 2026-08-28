package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
)

func TestRegexConstraints_IsAllowed(t *testing.T) {
	tests := map[string]struct {
		constraints config.RegexConstraints
		givenValue  string
		isAllowed   bool
	}{
		"should match all except ignored ones": {
			constraints: config.RegexConstraints{Include: []string{".*"}, Exclude: []string{"demo", "abc"}},
			givenValue:  "demo",
			isAllowed:   false,
		},
		"should allow non-excluded value when only exclude is set": {
			constraints: config.RegexConstraints{Exclude: []string{"demo"}},
			givenValue:  "other",
			isAllowed:   true,
		},
		"should ignore excluded value when only exclude is set": {
			constraints: config.RegexConstraints{Exclude: []string{"demo"}},
			givenValue:  "demo",
			isAllowed:   false,
		},
		"should ignore value matched by regex when only exclude is set": {
			constraints: config.RegexConstraints{Exclude: []string{"my-.*"}},
			givenValue:  "my-pod",
			isAllowed:   false,
		},
		"should only match included values": {
			constraints: config.RegexConstraints{Include: []string{"demo"}},
			givenValue:  "other",
			isAllowed:   false,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			actual, err := test.constraints.IsAllowed(test.givenValue)
			require.NoError(t, err)
			require.Equal(t, test.isAllowed, actual)
		})
	}
}
