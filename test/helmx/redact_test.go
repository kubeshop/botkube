package helmx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactAPIKey(t *testing.T) {

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "redact api key",
			input:    []string{"config.provider.apiKey=key:12345678-1234-5678-abcd-123456789abc"},
			expected: []string{"config.provider.apiKey=REDACTED"},
		},
		{
			name:     "redact api key with spaces",
			input:    []string{"config.provider.apiKey   =    key:abcdef12-3456-789a-bcde-abcdef123456"},
			expected: []string{"config.provider.apiKey   =REDACTED"},
		},
		{
			name:     "redact api key with multiple keys",
			input:    []string{"some other text", "another key=value", "config.provider.apiKey=key:98765432-4321-8765-dcba-987654321abc key=test"},
			expected: []string{"some other text", "another key=value", "config.provider.apiKey=REDACTED key=test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := redactAPIKey(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}
