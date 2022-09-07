package format

import (
	"fmt"
	"strings"
)

// CodeBlock trims whitespace and wraps a message in a code block.
func CodeBlock(msg string) string {
	return fmt.Sprintf("```\n%s\n```", strings.TrimSpace(msg))
}

// AdaptiveCodeBlock trims whitespace and wraps a message in a code block.
// If message is a single line, an inline code block is used.
func AdaptiveCodeBlock(msg string) string {
	code := func(in string) string {
		return fmt.Sprintf("`%s`", in)
	}
	if strings.Contains(msg, "\n") {
		code = func(in string) string {
			return fmt.Sprintf("```\n%s\n```", in)
		}
	}
	return code(strings.TrimSpace(msg))
}
