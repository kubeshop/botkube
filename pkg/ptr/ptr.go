package ptr

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

// IsTrue returns true if the given pointer is not nil and its value is true.
func IsTrue(in *bool) bool {
	if in == nil {
		return false
	}

	return *in
}
