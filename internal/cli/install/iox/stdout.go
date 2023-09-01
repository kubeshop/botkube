package iox

import (
	"os"
	"strings"

	"github.com/gookit/color"
)

// IndentFileWriter adds configured indent for messages starting with configured prefix.
type IndentFileWriter struct {
	triggerPrefix string
	indent        int
	w             *os.File
}

// NewIndentStdoutWriter returns a new IndentFileWriter instance.
func NewIndentStdoutWriter(triggerPrefix string, indent int) *IndentFileWriter {
	return NewIndentFileWriter(os.Stdout, triggerPrefix, indent)
}

// NewIndentFileWriter returns a new IndentFileWriter instance.
func NewIndentFileWriter(w *os.File, triggerPrefix string, indent int) *IndentFileWriter {
	return &IndentFileWriter{triggerPrefix: triggerPrefix, indent: indent, w: w}
}

// Fd returns the integer Unix file descriptor referencing the open file.
func (s *IndentFileWriter) Fd() uintptr {
	return s.w.Fd()
}

// Write writes len(b) bytes from b to a given os.File.
func (s *IndentFileWriter) Write(p []byte) (n int, err error) {
	if strings.HasPrefix(color.ClearCode(string(p)), s.triggerPrefix) {
		// we add indent only to messages that starts with a known prefix
		// as a result we don't alter messages which are terminal special codes, e.g. to clear the screen.
		_, err := s.w.Write([]byte(strings.Repeat(" ", s.indent)))
		if err != nil {
			return 0, err
		}
	}

	return s.w.Write(p)
}
