package loggerx

import (
	"os"

	"github.com/sirupsen/logrus"
)

// ExitOnError exits an app with a given error.
func ExitOnError(err error, context string) {
	if err == nil {
		return
	}
	log := &logrus.Logger{
		Out:          os.Stdout,
		Formatter:    &logrus.TextFormatter{FullTimestamp: true},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	log.Fatalf("%s: %s", context, err)
}
