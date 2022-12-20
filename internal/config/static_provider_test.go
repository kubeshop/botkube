package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticProviderSuccess(t *testing.T) {
	// when
	p := NewStaticProvider([]string{"/tmp/a.yaml"})
	configs, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	assert.Equal(t, []string{"/tmp/a.yaml"}, configs)
}
