package formatx

import (
	"fmt"
	"strings"
)

// BulletPointListFromMessages creates a bullet-point list from messages.
func BulletPointListFromMessages(msgs []string) string {
	if len(msgs) == 0 {
		return ""
	}
	return fmt.Sprintf("• %s", strings.Join(msgs, "\n• "))
}
