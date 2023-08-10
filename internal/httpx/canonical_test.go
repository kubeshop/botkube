package httpx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/httpx"
)

func TestCanonicalURLPath(t *testing.T) {
	tests := map[string]struct {
		givenPath string
		expPath   string
	}{
		"no trailing slash": {
			givenPath: "https://api.github.com",
			expPath:   "https://api.github.com/",
		},
		"multiple trailing slashes": {
			givenPath: "https://api.github.com///////////////",
			expPath:   "https://api.github.com/",
		},
		"single trailing slash": {
			givenPath: "https://api.github.com/",
			expPath:   "https://api.github.com/",
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// when
			normalizedPath := httpx.CanonicalURLPath(tc.givenPath)

			// then
			assert.Equal(t, tc.expPath, normalizedPath)
		})
	}
}
