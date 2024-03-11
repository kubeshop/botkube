package loggerx

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

// New returns a new logger based on a given configuration.
// It logs to stdout by default. It's a helper function to maintain backward compatibility.
func New(cfg config.Logger) logrus.FieldLogger {
	return NewStdout(cfg)
}

// NewStderr returns a new logger based on a given configuration. It logs to stderr.
func NewStderr(cfg config.Logger) logrus.FieldLogger {
	return newWithOutput(cfg, os.Stderr)
}

// NewStdout returns a new logger based on a given configuration. It logs to stdout.
func NewStdout(cfg config.Logger) logrus.FieldLogger {
	return newWithOutput(cfg, os.Stdout)
}

func newWithOutput(cfg config.Logger, output io.Writer) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetOutput(output)

	// Only logger the warning severity or above.
	logLevel, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		// Set Info level as a default
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	if cfg.Formatter == config.FormatterJSON {
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
