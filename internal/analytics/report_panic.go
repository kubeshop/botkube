package analytics

import (
	"fmt"
	"runtime/debug"
)

// FatalErrorAnalyticsReporter reports a fatal errors.
type FatalErrorAnalyticsReporter interface {
	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// ReportPanicLogger is a fakeLogger interface used by ReportPanicIfOccurs function.
type ReportPanicLogger interface {
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
}

// ReportPanicIfOccurs recovers from a panic and reports it, and then calls log.Fatal.
// This function should be called with `defer` at the beginning of the goroutine logic.
//
// NOTE: Make sure the reporter is not closed before reporting the panic. It will be cleaned up as a part of this function.
func ReportPanicIfOccurs(logger ReportPanicLogger, reporter FatalErrorAnalyticsReporter) {
	r := recover()
	if r == nil {
		return
	}

	panicDetailsErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
	err := reporter.ReportFatalError(panicDetailsErr)
	if err != nil {
		logger.Errorf("while reporting fatal error: %s", err.Error())
	}

	// Close the reader manually before exiting the app as it won't be cleaned up in other way.
	closeErr := reporter.Close()
	if closeErr != nil {
		logger.Errorf("while closing the reporter: %s", closeErr.Error())
	}

	// No other option than exiting the app
	logger.Fatal(panicDetailsErr)
}
