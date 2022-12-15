package loggerx

import (
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// NewNoop returns a logger that discards all logged messages.
// It's suitable for unit-tests.
func NewNoop() logrus.FieldLogger {
	logger, _ := logtest.NewNullLogger()
	return logger
}
