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
