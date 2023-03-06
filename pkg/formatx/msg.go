package formatx

import (
	"fmt"
	"strings"
)

// JoinMessages joins strings in slice with new line characters. It also appends a trailing newline at the end of message.
func JoinMessages(msgs []string) string {
	return strings.Join(msgs, "\n")
}

// BulletPointListFromMessages creates a bullet-point list from messages.
func BulletPointListFromMessages(msgs []string) string {
	if len(msgs) == 0 {
		return ""
	}
	return fmt.Sprintf("• %s", strings.Join(msgs, "\n• "))
}
