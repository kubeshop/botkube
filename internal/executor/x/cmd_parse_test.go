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
			input: "exec run helm list -A",
			expected: Command{
				ToExecute:     "exec run helm list -A",
				IsRawRequired: false,
			},
		},
		{
			input: "exec run helm list -A @raw",
			expected: Command{
				ToExecute:     "exec run helm list -A",
				IsRawRequired: true,
			},
		},
		{
			input: "exec run kubectl get pods @idx:123",
			expected: Command{
				ToExecute:     "exec run kubectl get pods",
				IsRawRequired: false,
			},
		},
		{
			input: "exec run kubectl get pods @idx:abc @page:12",
			expected: Command{
				ToExecute:     "exec run kubectl get pods @idx:abc",
				IsRawRequired: false,
				PageIndex:     12,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			// when
			gotCmd := Parse(tc.input)

			assert.Equal(t, tc.expected.ToExecute, gotCmd.ToExecute)
			assert.Equal(t, tc.expected.IsRawRequired, gotCmd.IsRawRequired)
			assert.Equal(t, tc.expected.PageIndex, gotCmd.PageIndex)
		})
	}
}
