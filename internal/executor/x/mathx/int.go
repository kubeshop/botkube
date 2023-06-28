package mathx

func IncreaseWithMax(in, max int) int {
	in++
	if in > max {
		return max
	}
	return in
}

func DecreaseWithMin(in, min int) int {
	in--
	if in < min {
		return min
	}
	return in
}
