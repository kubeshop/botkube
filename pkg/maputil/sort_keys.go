package maputil

import "sort"

func SortKeys[T any](in map[string]T) []string {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
