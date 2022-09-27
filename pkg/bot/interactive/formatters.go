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
