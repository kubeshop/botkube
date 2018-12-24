package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func init() {
	// Output to stdout instead of the default stderr
	Logger.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	Logger.SetLevel(logrus.DebugLevel)
	Logger.Formatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}
}
