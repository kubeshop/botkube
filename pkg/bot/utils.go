package bot

import (
	"fmt"
	"strings"
)

func formatCodeBlock(msg string) string {
	return fmt.Sprintf("```\n%s\n```", strings.TrimSpace(msg))
}
