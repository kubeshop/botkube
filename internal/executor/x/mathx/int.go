package mathx

// IncreaseWithMax increase in by 1 but only up to max value.
func IncreaseWithMax(in, max int) int {
	in++
	if in > max {
		return max
	}
	return in
}

// DecreaseWithMin decreases in by 1 but only to min value.
func DecreaseWithMin(in, min int) int {
	in--
	if in < min {
		return min
	}
	return in
}
