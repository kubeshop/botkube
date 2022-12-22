package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticProviderSuccess(t *testing.T) {
	// when
	p := NewFileSystemProvider([]string{"testdata/TestStaticProviderSuccess/config.yaml"})
	configs, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	content, err := os.ReadFile("testdata/TestStaticProviderSuccess/config.yaml")
	assert.NoError(t, err)
	assert.Equal(t, content, configs[0])
}

func TestSortCfgFiles(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
	}{
		"No special files": {
			input:    []string{"config.yaml", ".bar.yaml", "/_foo/bar.yaml", "/_bar/baz.yaml"},
			expected: []string{"config.yaml", ".bar.yaml", "/_foo/bar.yaml", "/_bar/baz.yaml"},
		},
		"Special files": {
			input:    []string{"_test.yaml", "config.yaml", "_foo.yaml", ".bar.yaml", "/bar/_baz.yaml"},
			expected: []string{"config.yaml", ".bar.yaml", "_test.yaml", "_foo.yaml", "/bar/_baz.yaml"},
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			actual := sortCfgFiles(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
