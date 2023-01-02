package helm

import (
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// TestCommand holds possible test options such as positional arguments and supported flags.
// Syntax:
//
//	helm test [RELEASE] [flags]
type TestCommand struct {
	noopValidator

	Name string `arg:"positional"`

	SupportedTestFlags
}

// Help returns command help message.
func (TestCommand) Help() string {
	return heredoc.Docf(`
		Runs the tests for a release.

		The argument this command takes is the name of a deployed release.
		The tests to be run are defined in the chart that was installed.

		Usage:
		  helm test [RELEASE] [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedTestFlags{}), 4))
}

// SupportedTestFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedTestFlags struct {
	Logs    bool          `arg:"--logs"`
	Timeout time.Duration `arg:"--timeout"`
}
