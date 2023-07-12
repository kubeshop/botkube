package iox

import (
	"os"
	"strings"

	"github.com/gookit/color"
)

// IndentStdoutWriter adds configured indent for messages starting with configured prefix.
type IndentStdoutWriter struct {
	triggerPrefix string
	indent        int
}

// NewIndentStdoutWriter returns a new IndentStdoutWriter instance.
func NewIndentStdoutWriter(triggerPrefix string, indent int) *IndentStdoutWriter {
	return &IndentStdoutWriter{triggerPrefix: triggerPrefix, indent: indent}
}

// Fd returns the integer Unix file descriptor referencing the open file.
func (s *IndentStdoutWriter) Fd() uintptr {
	return os.Stdout.Fd()
}

// Write writes len(b) bytes from b to os.Stdout.
func (s *IndentStdoutWriter) Write(p []byte) (n int, err error) {
	if strings.HasPrefix(color.ClearCode(string(p)), s.triggerPrefix) {
		// we add indent only to messages that starts with a known prefix
		// as a result we don't alter messages which are terminal special codes, e.g. to clear the screen.
		_, err := os.Stdout.Write([]byte(strings.Repeat(" ", s.indent)))
		if err != nil {
			return 0, err
		}
	}

	return os.Stdout.Write(p)
}
