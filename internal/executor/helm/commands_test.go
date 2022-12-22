package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveVersionFlag(t *testing.T) {
	tests := []struct {
		name      string
		givenArgs []string
		expArgs   []string
	}{

		{
			name:      "no givenArgs",
			givenArgs: []string{},
			expArgs:   []string{},
		},
		{
			name:      "no version flag provided",
			givenArgs: []string{"test"},
			expArgs:   []string{"test"},
		},
		{
			name:      "version flag without value",
			givenArgs: []string{"test", "--version"},
			expArgs:   []string{"test"},
		},
		{
			name:      "version flag with value after = sign",
			givenArgs: []string{"test", "--version=1.2.3"},
			expArgs:   []string{"test"},
		},
		{
			name:      "version flag with value in next argument",
			givenArgs: []string{"test", "--version", "1.2.3"},
			expArgs:   []string{"test"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs := removeVersionFlag(tc.givenArgs)
			assert.Equal(t, tc.expArgs, gotArgs)
		})
	}
}
