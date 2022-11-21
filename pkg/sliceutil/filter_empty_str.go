package sliceutil

// FilterEmptyStrings returns slice without empty strings.
func FilterEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		if len(s) == 0 {
			continue
		}
		result = append(result, s)
	}
	return result
}
