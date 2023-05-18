package printer

import (
	"fmt"
	"io"
)

// StaticSpinner is suitable for non-smart terminals.
type StaticSpinner struct {
	w      io.Writer
	active bool
}

// NewStaticSpinner returns a new StaticSpinner instance.
func NewStaticSpinner(w io.Writer) *StaticSpinner {
	return &StaticSpinner{w: w}
}

// Start activates the spinner with a given name.
func (s *StaticSpinner) Start(stage string) {
	s.active = true
	fmt.Fprintf(s.w, " • %s\n", stage)
}

// Active returns whether the spinner is currently active.
func (s *StaticSpinner) Active() bool {
	return s.active
}

// Stop deactivates the spinner with a given message.
func (s *StaticSpinner) Stop(msg string) {
	s.active = false
	fmt.Fprintln(s.w, msg)
}
