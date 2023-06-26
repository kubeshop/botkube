package migrate

import (
	"io"
	"syscall"

	"github.com/muesli/reflow/indent"
)

// IndentWriter provides functionality to intercept the underlying writer and add left indention.
type IndentWriter struct {
	w      io.Writer
	indent uint
}

// NewIndentWriter returns a new IndentWriter.
func NewIndentWriter(w io.Writer, indent uint) *IndentWriter {
	return &IndentWriter{w: w, indent: indent}
}

// Writer writes the input p.
func (e IndentWriter) Write(p []byte) (int, error) {
	n, err := e.w.Write(indent.Bytes(p, e.indent))
	if err != nil {
		return n, err
	}
	return len(p), nil
}

// Fd returns the integer Unix file descriptor referencing the open file.
func (e IndentWriter) Fd() uintptr {
	return uintptr(syscall.Stdout)
}
