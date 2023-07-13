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
	if cfg.Formatter == config.FormatterJson {
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		logger.Formatter = &logrus.TextFormatter{FullTimestamp: true, DisableColors: cfg.DisableColors, ForceColors: true}
	}

	return logger
}

// ExitOnError exits an app with a given error.
func ExitOnError(err error, context string) {
	if err == nil {
		return
	}
	log := &logrus.Logger{
		Out:          os.Stdout,
		Formatter:    &logrus.JSONFormatter{},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	log.Fatalf("%s: %s", context, err)
}
