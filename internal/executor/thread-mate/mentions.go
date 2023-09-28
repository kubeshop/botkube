package thread_mate

import (
	"fmt"
	"strings"
)

func extractIDFromMention(in string) string {
	in = strings.TrimPrefix(in, "<@")
	in = strings.TrimSuffix(in, ">")
	return in
}

func asMention(in string) string {
	return fmt.Sprintf("<@%s>", in)
}
