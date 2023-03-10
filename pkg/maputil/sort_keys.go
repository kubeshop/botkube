package maputil

import (
	"sort"

	"golang.org/x/exp/maps"
)

func SortKeys[T any](in map[string]T) []string {
	keys := maps.Keys(in)
	sort.Strings(keys)
	return keys
}
