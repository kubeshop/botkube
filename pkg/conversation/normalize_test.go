package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeChannelName(t *testing.T) {
	// given
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no hash",
			input:    "test",
			expected: "test",
		},
		{
			name:     "hash",
			input:    "#test",
			expected: "test",
		},
		{
			name:     "hash 2",
			input:    "###test",
			expected: "test",
		},
		{
			name:     "Discord",
			input:    "#1045026984801091655",
			expected: "1045026984801091655",
		},
		{
			name:     "whitespace 1",
			input:    " test ",
			expected: "test",
		},
		{
			name:     "whitespace 2",
			input:    " #test ",
			expected: "test",
		},
		{
			name:     "whitespace between (Mattermost)",
			input:    " Town Square ",
			expected: "Town Square",
		},
		{
			name:     "whitespace between (Mattermost) 2",
			input:    "Town Square",
			expected: "Town Square",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, changed := NormalizeChannelIdentifier(tc.input)

			// then
			assert.Equal(t, tc.expected, got)
			assert.Equal(t, tc.expected != tc.input, changed)
		})
	}
}
