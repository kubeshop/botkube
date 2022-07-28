package format

import (
	"fmt"
	"strings"
)

// CodeBlock trims whitespace and wraps a message in a code block.
func CodeBlock(msg string) string {
	return fmt.Sprintf("```\n%s\n```", strings.TrimSpace(msg))
}
