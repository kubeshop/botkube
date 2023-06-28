package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Command
	}{
		{
			input: "x run helm list -A",
			expected: Command{
				ToExecute:     "x run helm list -A",
				IsRawRequired: false,
			},
		},
		{
			input: "x run helm list -A @raw",
			expected: Command{
				ToExecute:     "x run helm list -A",
				IsRawRequired: true,
			},
		},
		{
			input: "x run kubectl get pods @idx:123",
			expected: Command{
				ToExecute:     "x run kubectl get pods",
				IsRawRequired: false,
			},
		},
		{
			input: "x run kubectl get pods @idx:abc",
			expected: Command{
				ToExecute:     "x run kubectl get pods @idx:abc @page:1",
				IsRawRequired: false,
				PageIndex:     1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			// when
			gotCmd := Parse(tc.input)

			assert.Equal(t, tc.expected.ToExecute, gotCmd.ToExecute)
			assert.Equal(t, tc.expected.IsRawRequired, gotCmd.IsRawRequired)
		})
	}
}
