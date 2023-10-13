// Package thread_mate here we define functions to help with platform mentions. For now, it covers only Slack syntax.
// In order to cover other platforms, we need to move it to the Botkube platform renderers and provide a common function
// that we can use in our message templating. For example,
//
//	body:
//	 plaintext: |
//	   Thanks for reaching out! Today, {{ .Assignee.ID | toMention }} will assist you in getting your Botkube up and running.
//
// This way `toMention` function can be properly set per each platform.
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
