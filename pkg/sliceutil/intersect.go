package sliceutil

import (
	"strings"
)

// Intersect checks whether two string slices have any common values.
func Intersect(this, that []string) bool {
	for _, i := range this {
		for _, j := range that {
			if strings.EqualFold(i, j) {
				return true
			}
		}
	}
	return false
}
