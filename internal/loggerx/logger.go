package loggerx

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

// New returns a new logger based on a given configuration.
func New(cfg config.Logger) logrus.FieldLogger {
	logger := logrus.New()
	// Output to stdout instead of the default stderr
	logger.SetOutput(os.Stdout)

	// Only logger the warning severity or above.
	logLevel, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		// Set Info level as a default
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true, DisableColors: cfg.DisableColors}

	return logger
}
