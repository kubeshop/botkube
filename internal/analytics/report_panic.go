package analytics

import (
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// FatalErrorAnalyticsReporter reports a fatal errors.
type FatalErrorAnalyticsReporter interface {
	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// ReportPanicIfOccurs recovers from a panic and reports it, and then calls log.Fatal.
// This function should be called with `defer` at the beginning of the goroutine logic.
//
// NOTE: Make sure the reporter is not closed before reporting the panic. It will be cleaned up as a part of this function.
func ReportPanicIfOccurs(log logrus.FieldLogger, reporter FatalErrorAnalyticsReporter) {
	r := recover()
	if r == nil {
		return
	}

	panicDetailsErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
	err := reporter.ReportFatalError(panicDetailsErr)
	if err != nil {
		log.WithError(err).Debug("failed to report panic for analytics")
	}

	// Close the reader manually before exiting the app as it won't be cleaned up in other way.
	closeErr := reporter.Close()
	if closeErr != nil {
		log.WithError(closeErr).Debug("failed to close analytics reporter")
	}

	// No other option than exiting the app
	log.Fatal(panicDetailsErr)
}
