package ptr

import "golang.org/x/exp/constraints"

// ToSlice converts a given slice of pointers to slice with non-nil elems.
func ToSlice[T any](in []*T) []T {
	var out []T
	for _, s := range in {
		if s == nil {
			continue
		}
		out = append(out, *s)
	}
	return out
}

// FromSlice returns slice of pointers for a given type.
func FromSlice[T any](in []T) []*T {
	out := make([]*T, 0, len(in))
	for idx := range in {
		out = append(out, &in[idx])
	}
	return out
}

// FromType returns pointer for a given input type.
func FromType[T any](in T) *T {
	return &in
}

// Primitives is a constraint that permits any primitive Go types.
type Primitives interface {
	constraints.Complex |
		constraints.Signed |
		constraints.Unsigned |
		constraints.Integer |
		constraints.Float |
		constraints.Ordered | ~bool
}

// ToValue returns value for a given pointer input type.
func ToValue[T Primitives](in *T) T {
	var empty T
	if in == nil {
		return empty
	}
	return *in
}

// AreAllSet returns true if all input strings are not nil and not empty.
func AreAllSet(in ...*string) bool {
	for _, item := range in {
		if item == nil || *item == "" {
			return false
		}
	}

	return true
}
