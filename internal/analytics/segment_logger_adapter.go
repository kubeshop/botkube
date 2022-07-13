package analytics

import (
	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
)

var _ segment.Logger = &segmentLoggerAdapter{}

type segmentLoggerAdapter struct {
	log logrus.FieldLogger
}

// NewSegmentLoggerAdapter returns new Segment logger adapter for logrus.FieldLogger.
func NewSegmentLoggerAdapter(log logrus.FieldLogger) segment.Logger {
	return &segmentLoggerAdapter{log: log}
}

func (l *segmentLoggerAdapter) Logf(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

func (l *segmentLoggerAdapter) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}
