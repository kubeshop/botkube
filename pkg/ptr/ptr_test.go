package ptr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/ptr"
)

func TestToSlice(t *testing.T) {
	tests := []struct {
		name  string
		given []*string
		exp   []string
	}{
		{
			name:  "Test with all nil items",
			given: []*string{nil, nil},
			exp:   nil,
		},
		{
			name:  "Test with some nil items",
			given: []*string{nil, ptr.FromType("elem2"), nil, ptr.FromType("elem3")},
			exp:   []string{"elem2", "elem3"},
		},
		{
			name:  "Test without nil items",
			given: []*string{ptr.FromType("elem1"), ptr.FromType("elem2"), ptr.FromType("elem3")},
			exp:   []string{"elem1", "elem2", "elem3"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := ptr.ToSlice(tc.given)
			// then
			assert.Equal(t, tc.exp, got)
		})
	}
}

func TestFromType(t *testing.T) {
	type exampleStruct struct {
		Name string
	}
	tests := []struct {
		name  string
		given any
	}{
		{
			name:  "Test with number",
			given: 1,
		},
		{
			name:  "Test with string",
			given: "test",
		},
		{
			name:  "Test with bool",
			given: true,
		},
		{
			name: "Test with struct",
			given: exampleStruct{
				Name: "test",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := ptr.FromType(tc.given)

			// then
			assert.NotNil(t, got)
			assert.EqualValues(t, tc.given, *got)
		})
	}
}

func TestToValue(t *testing.T) {
	t.Run("Test with number", func(t *testing.T) {
		given := ptr.FromType(1)
		got := ptr.ToValue(given)
		assert.EqualValues(t, *given, got)
	})
	t.Run("Test with string", func(t *testing.T) {
		given := ptr.FromType("test")
		got := ptr.ToValue(given)
		assert.EqualValues(t, *given, got)
	})
	t.Run("Test with bool", func(t *testing.T) {
		given := ptr.FromType(true)
		got := ptr.ToValue(given)
		assert.EqualValues(t, *given, got)
	})
	t.Run("Test with nil", func(t *testing.T) {
		given := ptr.ToValue[bool](nil)
		assert.False(t, given)
	})
}
