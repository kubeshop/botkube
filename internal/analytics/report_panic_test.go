package analytics_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/analytics"
)

func TestReportPanicIfOccurs_Panic(t *testing.T) {
	// given
	testCases := []struct {
		Name                            string
		FnToRun                         func()
		InputReporter                   *fakeReporter
		ExpectedErrMessageSubstr        string
		ExpectedReportErrMessage        string
		ExpectedReporterCloseErrMessage string
	}{
		{
			Name: "Success panic report",
			FnToRun: func() {
				panic("foo")
			},
			InputReporter:            &fakeReporter{},
			ExpectedErrMessageSubstr: "panic: foo",
		},
		{
			Name: "Error during reporting panic",
			FnToRun: func() {
				panic("foo")
			},
			InputReporter:            &fakeReporter{closed: true},
			ExpectedErrMessageSubstr: "panic: foo",
			ExpectedReportErrMessage: "while reporting fatal error: reporter shouldn't be closed",
		},
		{
			Name: "Error when closing reporter",
			FnToRun: func() {
				panic("foo")
			},
			InputReporter:                   &fakeReporter{shouldReturnCloseErr: true},
			ExpectedErrMessageSubstr:        "panic: foo",
			ExpectedReporterCloseErrMessage: "while closing the reporter: closing error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			log := &fakeLogger{}
			reporter := tc.InputReporter

			testFunc := func() {
				defer analytics.ReportPanicIfOccurs(log, reporter)

				panic("foo")
			}

			// when
			testFunc()

			// then

			// log.Fatal triggered regardless the reporting status
			require.Len(t, log.fatalReported, 1)
			fatalErr, ok := log.fatalReported[0].(error)
			require.True(t, ok)
			require.NotNil(t, fatalErr)
			assert.Contains(t, fatalErr.Error(), tc.ExpectedErrMessageSubstr)

			// See if a report error was logged (if occurred)
			if tc.ExpectedReportErrMessage != "" {
				require.Len(t, log.errReported, 1)
				assert.Equal(t, tc.ExpectedReportErrMessage, log.errReported[0])
				return
			}

			// Panic reported (if reporter didn't return error)
			require.NotNil(t, reporter.reportedErr)
			assert.Contains(t, reporter.reportedErr.Error(), tc.ExpectedErrMessageSubstr)
			assert.True(t, reporter.closed)

			// See if the reporter close error was logged (if occurred)
			if tc.ExpectedReporterCloseErrMessage != "" {
				require.Len(t, log.errReported, 1)
				assert.Equal(t, tc.ExpectedReporterCloseErrMessage, log.errReported[0])
				return
			}

			// No errors logged in happy path scenario
			assert.Empty(t, log.errReported)
		})
	}
}

func TestReportPanicIfOccurs_NoPanic(t *testing.T) {
	//given
	log := &fakeLogger{}
	reporter := &fakeReporter{shouldReturnCloseErr: true, closed: true} // fails when calling any method

	testFunc := func() {
		defer analytics.ReportPanicIfOccurs(log, reporter)
	}

	// when
	testFunc()

	// then
	assert.Empty(t, log.fatalReported)
	assert.Empty(t, log.errReported)
	assert.Empty(t, reporter.reportedErr)
}

type fakeReporter struct {
	shouldReturnCloseErr bool
	reportedErr          error
	closed               bool
}

func (r *fakeReporter) ReportFatalError(err error) error {
	if r.closed {
		return errors.New("reporter shouldn't be closed")
	}
	r.reportedErr = err
	return nil
}

func (r *fakeReporter) Close() error {
	r.closed = true
	if r.shouldReturnCloseErr {
		return errors.New("closing error")
	}
	return nil
}

type fakeLogger struct {
	fatalReported []interface{}
	errReported   []string
}

func (l *fakeLogger) Errorf(format string, args ...interface{}) {
	l.errReported = append(l.errReported, fmt.Sprintf(format, args...))
}

func (l *fakeLogger) Fatal(args ...interface{}) {
	l.fatalReported = append(l.fatalReported, args...)
}
