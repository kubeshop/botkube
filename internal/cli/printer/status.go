package printer

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/kubeshop/botkube/internal/cli"
)

// Spinner defines interface for terminal spinner.
type Spinner interface {
	Start(stage string)
	Active() bool
	Stop(msg string)
}

// Status defines status printer methods. Allows us to use different status printers.
type Status interface {
	Step(stageFmt string, args ...interface{})
	End(success bool)
	Infof(format string, a ...interface{})
	InfoWithBody(header, body string)
	Writer() io.Writer
}

// StatusPrinter provides functionality to display steps progress in terminal.
type StatusPrinter struct {
	w io.Writer

	spinner         Spinner
	durationSprintf func(format string, a ...interface{}) string

	timeStarted time.Time
	stage       string
}

// NewStatus returns a new Status instance.
func NewStatus(w io.Writer, header string) *StatusPrinter {
	if header != "" {
		fmt.Fprintln(w, header)
	}

	st := &StatusPrinter{
		w: w,
	}
	if cli.IsSmartTerminal(w) {
		st.durationSprintf = color.New(color.Faint, color.Italic).Sprintf
		st.spinner = NewDynamicSpinner(w)
	} else {
		st.durationSprintf = fmt.Sprintf
		st.spinner = NewStaticSpinner(w)
	}

	return st
}

// Step starts spinner for a given step.
func (s *StatusPrinter) Step(stageFmt string, args ...interface{}) {
	// Finish previously started step
	s.End(true)

	s.timeStarted = time.Now()
	started := fmt.Sprintf(" [started %s]", s.timeStarted.Format("15:04 MST"))

	s.stage = fmt.Sprintf(stageFmt, args...)
	msg := fmt.Sprintf("%s%s", s.stage, started)
	s.spinner.Start(msg)
}

// End marks started step as completed.
func (s *StatusPrinter) End(success bool) {
	if !s.spinner.Active() {
		return
	}

	var icon string
	if success {
		icon = color.GreenString("✓")
	} else {
		icon = color.RedString("✗")
	}

	dur := s.durationSprintf(" [took %s]", duration.HumanDuration(time.Since(s.timeStarted)))
	msg := fmt.Sprintf(" %s %s%s\n",
		icon, s.stage, dur)
	s.spinner.Stop(msg)
}

// Writer returns underlying io.Writer
func (s *StatusPrinter) Writer() io.Writer {
	return s.w
}

// Infof prints a given info without spinner animation.
func (s *StatusPrinter) Infof(format string, a ...interface{}) {
	// Ensure that previously started step is finished. Without that we will mess up our output.
	s.End(true)

	fmt.Fprintf(s.w, " • %s\n", fmt.Sprintf(format, a...))
}

// InfoWithBody prints a given info with a given body and without spinner animation.
func (s *StatusPrinter) InfoWithBody(header, body string) {
	// Ensure that previously started step is finished. Without that we will mess up our output.
	s.End(true)

	fmt.Fprintf(s.w, " • %s\n%s", header, body)
}
