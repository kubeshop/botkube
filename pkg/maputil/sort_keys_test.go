package maputil_test

import (
	"github.com/kubeshop/botkube/pkg/maputil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSortKeys(t *testing.T) {
	// given
	in := map[string]int{
		"b": 2,
		"a": 1,
		"c": 3,
		"d": 4,
		"e": 5,
	}
	expected := []string{"a", "b", "c", "d", "e"}

	// when
	out := maputil.SortKeys(in)

	// then
	assert.Equal(t, expected, out)
}
