package builder

import (
	"strings"
)

func overflowSentence(in []string) []string {
	for idx := range in {
		if len(in[idx]) < 76 { // Maximum length for text field in dropdown is 75 characters. (https://api.slack.com/reference/block-kit/composition-objects#option)
			continue
		}

		in[idx] = in[idx][:72] + "…"
	}
	return in
}

func getNonEmptyLines(in string) []string {
	lines := strings.FieldsFunc(in, splitByNewLines)
	var out []string
	for _, x := range lines {
		if x == "" {
			continue
		}
		out = append(out, x)
	}
	return out
}

func splitByNewLines(c rune) bool {
	return c == '\n' || c == '\r'
}
