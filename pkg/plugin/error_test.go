package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFoundPluginError(t *testing.T) {
	// when
	err := NewNotFoundPluginError("test")
	// then
	assert.True(t, IsNotFoundError(err))

	// when
	wrapped := fmt.Errorf("wrapping root error: %w", err)
	// then
	assert.True(t, IsNotFoundError(wrapped))
}
