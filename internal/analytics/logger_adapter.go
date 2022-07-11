package analytics

import "github.com/sirupsen/logrus"

type loggerAdapter struct {
	log logrus.FieldLogger
}

func newLoggerAdapter(log logrus.FieldLogger) *loggerAdapter {
	return &loggerAdapter{log: log}
}

func (l *loggerAdapter) Logf(format string, args ...interface{}) {
	l.log.Infof(format, args) // TODO: Should we use Debug?
}

func (l *loggerAdapter) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args)
}
