package formatx

import (
	"fmt"
	"strings"
)

// JoinMessages joins strings in slice with new line characters. It also appends a trailing newline at the end of message.
func JoinMessages(msgs []string) string {
	return joinMessages(msgs, "")
}

// BulletPointListFromMessages creates a Markdown bullet-point list from messages.
// See https://api.slack.com/reference/surfaces/formatting#block-formatting
func BulletPointListFromMessages(msgs []string) string {
	return joinMessages(msgs, "• ")
}

func joinMessages(msgs []string, msgPrefix string) string {
	if len(msgs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	for _, m := range msgs {
		strBuilder.WriteString(fmt.Sprintf("%s%s\n", msgPrefix, m))
	}

	return strBuilder.String()
}
