package sliceutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/sliceutil"
)

func TestIntersect_IsCaseInsensitive(t *testing.T) {
	assert.True(t, sliceutil.Intersect([]string{"a", "B"}, []string{"b"}))
	assert.True(t, sliceutil.Intersect([]string{"a", "B"}, []string{"A", "b"}))
	assert.False(t, sliceutil.Intersect([]string{"a", "B"}, []string{"c"}))
}
