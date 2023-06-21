package ptr

import "golang.org/x/exp/constraints"

// ToBool returns bool value for a given pointer.
func ToBool(in *bool) bool {
	if in == nil {
		return false
	}
	return *in
}

// Bool returns pointer to a given input bool value.
func Bool(in bool) *bool {
	return &in
}

// String returns pointer to a given input string value.
func String(in string) *string {
	return &in
}

// IsTrue returns true if the given pointer is not nil and its value is true.
func IsTrue(in *bool) bool {
	if in == nil {
		return false
	}

	return *in
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
