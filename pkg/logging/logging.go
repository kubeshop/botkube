package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger global object for logging across the pkg/
var Logger = logrus.New()

func init() {
	// Output to stdout instead of the default stderr
	Logger.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		// Set Info level as a default
		logLevel = logrus.InfoLevel
	}
	Logger.SetLevel(logLevel)
	Logger.Formatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}
}
