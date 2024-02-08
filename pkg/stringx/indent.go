package stringx

import (
	"strings"
)

// IndentAfterLine adds intent to a given string but only after a given line.
func IndentAfterLine(in string, afterLine int, indent string) string {
	lines := strings.FieldsFunc(in, splitByNewLines)
	if len(lines) < afterLine {
		return in
	}

	var out []string
	for idx, x := range lines {
		if idx+1 > afterLine {
			x = indent + x
		}
		out = append(out, x)
	}
	return strings.Join(out, "\n")
}

func splitByNewLines(c rune) bool {
	return c == '\n' || c == '\r'
}
