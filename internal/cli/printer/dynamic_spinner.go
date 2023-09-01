package printer

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// DynamicSpinner is suitable for smart terminals which support animations (control cursor location, color etc.).
type DynamicSpinner struct {
	underlying *spinner.Spinner
}

// NewDynamicSpinner returns a new DynamicSpinner instance.
func NewDynamicSpinner(w *os.File) *DynamicSpinner {
	// if dynamic is used, then the writer is smart, so color will be supported
	color.NoColor = false
	return &DynamicSpinner{
		underlying: spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithWriterFile(w)),
	}
}

// Start activates the spinner with a given name.
func (d *DynamicSpinner) Start(stage string) {
	d.underlying.Prefix = " "
	d.underlying.Suffix = fmt.Sprintf(" %s", stage)
	d.underlying.Start()
}

// Active returns whether the spinner is currently active.
func (d *DynamicSpinner) Active() bool {
	return d.underlying.Active()
}

// Stop deactivates the spinner with a given message.
func (d *DynamicSpinner) Stop(msg string) {
	d.underlying.FinalMSG = msg
	d.underlying.Stop()
}
