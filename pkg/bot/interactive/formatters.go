package interactive

import (
	"fmt"
)

// NewlineFormatter adds new line formatting.
func NewlineFormatter(msg string) string {
	return fmt.Sprintf("%s\n", msg)
}

// MdHeaderFormatter adds Markdown header formatting.
func MdHeaderFormatter(msg string) string {
	return fmt.Sprintf("**%s**", msg)
}

// NoFormatting does not apply any formatting.
func NoFormatting(msg string) string {
	return msg
}
