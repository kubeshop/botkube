package format

import (
	"fmt"
	"strings"
)

// JoinMessages joins strings in slice with new line characters. It also appends a trailing newline at the end of message.
func JoinMessages(msgs []string) string {
	if len(msgs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	for _, m := range msgs {
		strBuilder.WriteString(fmt.Sprintf("%s\n", m))
	}

	return strBuilder.String()
}
