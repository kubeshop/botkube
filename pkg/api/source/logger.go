package source

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger returns a new logger used internally. We should replace it in the near future, as we shouldn't be so opinionated.
func NewLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	return logger
}
